package shared

import (
	"errors"
	"testing"
)

func TestPaginationSanitizeOffsetAndSort(t *testing.T) {
	p := PaginationParams{Page: 0, Limit: 101}
	p.Sanitize()
	if p.Page != 1 || p.Limit != 10 || p.Offset() != 0 {
		t.Fatalf("Sanitize() = page %d, limit %d, offset %d", p.Page, p.Limit, p.Offset())
	}

	p = PaginationParams{Page: 3, Limit: 20, Sort: []SortParam{{Field: "created_at", Direction: SortDesc}}}
	if err := p.ValidateSort(map[string]bool{"created_at": true}); err != nil {
		t.Fatalf("ValidateSort() rejected allowed sort: %v", err)
	}
	if p.Offset() != 40 {
		t.Fatalf("Offset() = %d, want 40", p.Offset())
	}

	for _, sort := range []SortParam{{Field: "secret", Direction: SortAsc}, {Field: "created_at", Direction: "SIDEWAYS"}} {
		invalid := PaginationParams{Sort: []SortParam{sort}}
		if err := invalid.ValidateSort(map[string]bool{"created_at": true}); err == nil {
			t.Errorf("ValidateSort(%+v) succeeded", sort)
		}
	}
}

func TestFiltersValidationAndWithout(t *testing.T) {
	original := Filters{
		{Field: "expires_at", Operator: OperatorGreaterThan},
		{Field: "revoked_at", Operator: OperatorIsNull},
	}
	filtered := original.Without("expires_at")
	if len(filtered) != 1 || filtered[0].Field != "revoked_at" || len(original) != 2 {
		t.Fatalf("Without() = %+v; original = %+v", filtered, original)
	}
	if err := (Filter{Operator: OperatorILike}).Validate(); err != nil {
		t.Fatalf("Validate() rejected known operator: %v", err)
	}
	if err := (Filter{Operator: "DROP TABLE"}).Validate(); err == nil {
		t.Fatal("Validate() accepted unknown operator")
	}
	if err := original.ValidateAllowed(map[string]bool{"revoked_at": true, "expires_at": true}); err != nil {
		t.Fatalf("ValidateAllowed() rejected allowed fields: %v", err)
	}
	if err := original.ValidateAllowed(map[string]bool{"revoked_at": true}); err == nil {
		t.Fatal("ValidateAllowed() accepted an unknown field")
	}
	for _, tc := range []struct {
		operator FilterOperator
		set      bool
		null     bool
	}{
		{operator: OperatorIn, set: true},
		{operator: OperatorNotIn, set: true},
		{operator: OperatorIsNull, null: true},
		{operator: OperatorIsNotNull, null: true},
		{operator: OperatorEquals},
	} {
		filter := Filter{Operator: tc.operator}
		if filter.IsSetOperator() != tc.set || filter.IsNullOperator() != tc.null {
			t.Errorf("operator %q: IsSetOperator()=%v IsNullOperator()=%v", tc.operator, filter.IsSetOperator(), filter.IsNullOperator())
		}
	}
}

func TestImageFormatsAndAppErrorUnwrap(t *testing.T) {
	if _, ok := ImageFormatFromDecoder("gif"); ok {
		t.Fatal("ImageFormatFromDecoder accepted an unsupported format")
	}
	for key, extension := range map[string]string{"avatar.JPG": ".jpg", "avatar.jpeg": ".jpg", "avatar.PNG": ".png"} {
		format, ok := ImageFormatFromKey(key)
		if !ok || format.Extension != extension {
			t.Errorf("ImageFormatFromKey(%q) = %+v, %v", key, format, ok)
		}
	}
	if _, ok := ImageFormatFromKey("avatar.gif"); ok {
		t.Fatal("ImageFormatFromKey accepted unsupported format")
	}

	cause := errors.New("database down")
	appErr := NewAppError(ErrTypeInternal, "request failed", cause)
	if !errors.Is(appErr, cause) || appErr.Error() != "request failed: database down" {
		t.Fatalf("AppError did not preserve its cause: %v", appErr)
	}
	if got := NewAppError(ErrTypeNotFound, "missing", nil).Error(); got != "missing" {
		t.Fatalf("AppError without cause = %q, want plain message", got)
	}
}
