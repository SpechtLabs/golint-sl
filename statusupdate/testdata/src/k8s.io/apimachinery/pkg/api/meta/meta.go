// Package meta is a stub of k8s.io/apimachinery/pkg/api/meta for testing the
// statusupdate analyzer.
package meta

// Condition is a trimmed metav1.Condition.
type Condition struct {
	Type   string
	Status string
}

// SetStatusCondition sets the condition on the slice.
func SetStatusCondition(conditions *[]Condition, c Condition) {}

// RemoveStatusCondition removes the condition from the slice.
func RemoveStatusCondition(conditions *[]Condition, conditionType string) {}

// IsStatusConditionTrue reports whether the condition is true.
func IsStatusConditionTrue(conditions []Condition, conditionType string) bool { return false }
