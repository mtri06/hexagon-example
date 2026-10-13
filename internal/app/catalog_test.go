package app_test

import (
	"fmt"
	"net/http"
	"testing"

	"goschool/internal/domain/course"
	"goschool/internal/domain/derr"
	"goschool/internal/domain/subject"
)

func idOf(r response) int { return int(r.Body["id"].(float64)) }

func TestCatalog(t *testing.T) {
	srv, _ := setup(t)
	admin := newClient(t, srv.URL)
	admin.expect(admin.login("root", "rootpassword"), 200, "")
	student := newClient(t, srv.URL)
	student.expect(student.login("stu", "studentpass"), 200, "")

	// --- subjects ---
	student.expect(student.do(http.MethodPost, "/subjects", map[string]any{"name": "Math"}), 403, "forbidden")
	admin.expect(admin.do(http.MethodPost, "/subjects", map[string]any{}), 400, "invalid_body")
	admin.expect(admin.do(http.MethodPost, "/subjects", map[string]any{"name": "   "}), 400, derr.CodeValidationFailed)

	math := admin.do(http.MethodPost, "/subjects", map[string]any{"name": " Math "})
	admin.expect(math, 201, "")
	if math.Body["name"] != "Math" {
		t.Fatalf("name not trimmed: %v", math.Body)
	}
	mathID := idOf(math)
	admin.expect(admin.do(http.MethodPost, "/subjects", map[string]any{"name": "MATH"}), 409, subject.CodeSubjectNameAlreadyExists)
	lit := admin.do(http.MethodPost, "/subjects", map[string]any{"name": "Literature"})
	admin.expect(lit, 201, "")

	// Any role can read.
	list := student.do(http.MethodGet, "/subjects?q=mat", nil)
	student.expect(list, 200, "")
	if list.Body["meta"].(map[string]any)["total"].(float64) != 1 {
		t.Fatalf("subject search: %v", list.Body)
	}
	student.expect(student.do(http.MethodGet, fmt.Sprintf("/subjects/%d", mathID), nil), 200, "")
	student.expect(student.do(http.MethodGet, "/subjects/9999", nil), 404, subject.CodeSubjectNotFound)

	renamed := admin.do(http.MethodPatch, fmt.Sprintf("/subjects/%d", mathID), map[string]any{"name": "Mathematics"})
	admin.expect(renamed, 200, "")
	if renamed.Body["name"] != "Mathematics" {
		t.Fatalf("rename: %v", renamed.Body)
	}
	admin.expect(admin.do(http.MethodPatch, fmt.Sprintf("/subjects/%d", mathID), map[string]any{"name": "literature"}), 409, subject.CodeSubjectNameAlreadyExists)

	// --- courses ---
	student.expect(student.do(http.MethodPost, "/courses", map[string]any{"subject_id": mathID, "name": "Math 7"}), 403, "forbidden")
	admin.expect(admin.do(http.MethodPost, "/courses", map[string]any{"subject_id": 9999, "name": "Ghost"}), 404, subject.CodeSubjectNotFound)
	admin.expect(admin.do(http.MethodPost, "/courses", map[string]any{"subject_id": mathID, "name": "X", "status": "archived"}), 400, "invalid_body")

	m7 := admin.do(http.MethodPost, "/courses", map[string]any{"subject_id": mathID, "name": "Math 7"})
	admin.expect(m7, 201, "")
	if m7.Body["status"] != "active" || m7.Body["subject_name"] != "Mathematics" {
		t.Fatalf("course defaults: %v", m7.Body)
	}
	m7ID := idOf(m7)
	admin.expect(admin.do(http.MethodPost, "/courses", map[string]any{"subject_id": mathID, "name": "math 7"}), 409, course.CodeCourseNameAlreadyExists)
	m8 := admin.do(http.MethodPost, "/courses", map[string]any{"subject_id": mathID, "name": "Math 8", "status": "inactive"})
	admin.expect(m8, 201, "")
	admin.expect(admin.do(http.MethodPost, "/courses", map[string]any{"subject_id": idOf(lit), "name": "Literature 7"}), 201, "")

	byMath := student.do(http.MethodGet, fmt.Sprintf("/courses?subject_id=%d&status=active", mathID), nil)
	student.expect(byMath, 200, "")
	if data := byMath.Body["data"].([]any); len(data) != 1 || data[0].(map[string]any)["name"] != "Math 7" {
		t.Fatalf("course filters: %v", byMath.Body)
	}
	student.expect(student.do(http.MethodGet, "/courses?status=archived", nil), 400, derr.CodeValidationFailed)
	student.expect(student.do(http.MethodGet, "/courses?subject_id=abc", nil), 400, "invalid_query")

	admin.expect(admin.do(http.MethodPatch, fmt.Sprintf("/courses/%d", m7ID), map[string]any{}), 400, derr.CodeNothingToUpdate)
	upd := admin.do(http.MethodPatch, fmt.Sprintf("/courses/%d", m7ID), map[string]any{"status": "inactive", "subject_id": idOf(lit)})
	admin.expect(upd, 200, "")
	if upd.Body["status"] != "inactive" || int(upd.Body["subject_id"].(float64)) != mathID {
		t.Fatalf("status change / subject immutability: %v", upd.Body)
	}

	// --- deletes ---
	admin.expect(admin.do(http.MethodDelete, fmt.Sprintf("/subjects/%d", mathID), nil), 409, subject.CodeSubjectInUse)
	student.expect(student.do(http.MethodDelete, fmt.Sprintf("/courses/%d", m7ID), nil), 403, "forbidden")
	admin.expect(admin.do(http.MethodDelete, fmt.Sprintf("/courses/%d", m7ID), nil), 204, "")
	admin.expect(admin.do(http.MethodDelete, fmt.Sprintf("/courses/%d", idOf(m8)), nil), 204, "")
	admin.expect(admin.do(http.MethodGet, fmt.Sprintf("/courses/%d", m7ID), nil), 404, course.CodeCourseNotFound)
	admin.expect(admin.do(http.MethodDelete, fmt.Sprintf("/subjects/%d", mathID), nil), 204, "")
	admin.expect(admin.do(http.MethodDelete, fmt.Sprintf("/subjects/%d", mathID), nil), 404, subject.CodeSubjectNotFound)
}
