package helper

import (
	"fmt"
	"math"
	"net/url"
	"reflect"
	"strings"
	"unicode/utf8"
)

const MaxMoney = 1_000_000_000.0

// ValidateInputLimits applies shared upper bounds to decoded API inputs, including
// anonymous request structs and nested collections. Required/business rules remain
// with the endpoint and service validators. String lengths count Unicode runes.
func ValidateInputLimits(input any) error {
	if err := validateInputValue(reflect.ValueOf(input), "", "", 0); err != nil {
		return InvalidInput("%s", err)
	}
	return nil
}

func validateInputValue(v reflect.Value, name, parent string, depth int) error {
	if depth > 32 {
		return fmt.Errorf("input nesting exceeds 32 levels")
	}
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil
	}
	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			key := strings.Split(f.Tag.Get("json"), ",")[0]
			if key == "-" {
				continue
			}
			if key == "" {
				key = strings.Split(f.Tag.Get("query"), ",")[0]
			}
			if key == "" {
				key = strings.ToLower(f.Name)
			}
			if err := validateInputValue(v.Field(i), key, t.Name(), depth+1); err != nil {
				return err
			}
		}
	case reflect.String:
		s := v.String()
		min, max := 0, 0
		if parent == "CreateComplaintRequest" {
			switch name {
			case "subject":
				min, max = 3, 200
			case "body":
				min, max = 10, 5000
			}
		} else if parent == "ResolveComplaintRequest" && name == "admin_note" {
			min, max = 3, 2000
		}
		if min > 0 {
			n := utf8.RuneCountInString(strings.TrimSpace(s))
			if n < min || n > max {
				return fmt.Errorf("%s must contain between %d and %d characters after trimming whitespace", name, min, max)
			}
		}
		limit := 10000
		switch name {
		case "title":
			limit = 50
			if (parent == "UpdateProjectRequest" || parent == "UpdateMilestoneRequest") && strings.TrimSpace(s) == "" {
				return fmt.Errorf("title must not be blank")
			}
		case "description":
			limit = 5000
			if parent == "UpdateProjectRequest" {
				limit = 40
			}
		case "first_name", "last_name", "name", "name_th", "name_en", "faculty", "major", "bank_name", "account_name", "transfer_ref", "search":
			limit = 100
		case "email":
			limit = 254
		case "phone", "account_number", "student_code":
			limit = 30
		case "bio", "address", "reason", "note", "admin_note", "about", "place":
			limit = 2000
		case "acceptance_criteria", "summary", "message", "question", "answer":
			limit = 5000
		case "password", "old_password", "new_password":
			if len(s) > 72 {
				return fmt.Errorf("%s must not exceed 72 bytes", name)
			}
		}
		isURL := name == "evidence" || name == "url" || name == "urls" || name == "links" || name == "attachments" || name == "cover_image" || name == "picture" || name == "slip_image" || name == "link" || strings.HasSuffix(name, "_url")
		if isURL {
			limit = 2048
		}
		if !utf8.ValidString(s) || utf8.RuneCountInString(s) > limit {
			return fmt.Errorf("%s must be valid text of at most %d characters", name, limit)
		}
		if strings.ContainsRune(s, 0) {
			return fmt.Errorf("%s must not contain null characters", name)
		}
		if isURL && s != "" {
			u, err := url.Parse(s)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
				return fmt.Errorf("%s must be an HTTP or HTTPS URL", name)
			}
		}
	case reflect.Float32, reflect.Float64:
		x := v.Float()
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return fmt.Errorf("%s must be finite", name)
		}
		max := MaxMoney
		if strings.Contains(name, "pct") || name == "platform_fee" {
			max = 100
		}
		if x < 0 || x > max {
			return fmt.Errorf("%s must be between 0 and %g", name, max)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x := v.Int()
		max := int64(math.MaxInt32)
		switch name {
		case "duration":
			max = 365
		case "duration_days":
			max = 60
		case "duration_months":
			max = 48
		case "phase_no", "quarter_no":
			max = 4
		case "percent_release":
			max = 100
		}
		if x < 0 || x > max {
			return fmt.Errorf("%s must be between 0 and %d", name, max)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v.Uint() > math.MaxInt64 {
			return fmt.Errorf("%s exceeds the supported integer range", name)
		}
	case reflect.Slice, reflect.Array, reflect.Map:
		max := 100
		switch name {
		case "links":
			max = 5
		case "urls", "attachments", "type":
			max = 10
		case "milestones":
			max = 4
		case "criteria":
			max = 50
		}
		if v.Len() > max {
			return fmt.Errorf("%s must contain at most %d items", name, max)
		}
		if v.Kind() == reflect.Map {
			iter := v.MapRange()
			for iter.Next() {
				if err := validateInputValue(iter.Key(), "key", parent, depth+1); err != nil {
					return err
				}
				if err := validateInputValue(iter.Value(), name, parent, depth+1); err != nil {
					return err
				}
			}
		} else {
			for i := 0; i < v.Len(); i++ {
				if err := validateInputValue(v.Index(i), name, parent, depth+1); err != nil {
					return err
				}
			}
			if name == "urls" || name == "attachments" {
				urls := make([]string, 0, v.Len())
				for i := 0; i < v.Len(); i++ {
					if v.Index(i).Kind() == reflect.String {
						urls = append(urls, v.Index(i).String())
					}
				}
				if err := ValidateMediaURLs(urls); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
