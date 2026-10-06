package shared

// Nullable is a PATCH field that distinguishes "leave unchanged" (Set=false)
// from "clear" (Set=true, Value=nil) and "set" (Set=true, Value!=nil).
type Nullable[T any] struct {
	Set   bool
	Value *T
}

func SetTo[T any](v T) Nullable[T] { return Nullable[T]{Set: true, Value: &v} }
func SetNull[T any]() Nullable[T]  { return Nullable[T]{Set: true} }
