package helper

import (
	"errors"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()

	// Tanpa ini, pesan error nyebut nama field Go ("Username"),
	// padahal client ngirim dan baca nama JSON ("username").
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})

	// Custom validation 1: Nggak boleh ada spasi
	_ = v.RegisterValidation("nospace", func(fl validator.FieldLevel) bool {
		return !strings.ContainsAny(fl.Field().String(), " \t\n\r")
	})

	// Custom validation 2: Username hanya boleh huruf, angka, titik, underscore
	_ = v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		for _, r := range fl.Field().String() {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' && r != '_' {
				return false
			}
		}
		return true
	})

	// Custom validation 3: Password minimum 8 huruf (bisa ditambah cek kompleksitas)
	_ = v.RegisterValidation("strongpassword", func(fl validator.FieldLevel) bool {
		return len(fl.Field().String()) >= 8
	})

	// Custom validation 4: NIM (Harus 9 digit, diawali angka 4, semuanya angka)
	_ = v.RegisterValidation("nim", func(fl validator.FieldLevel) bool {
		val := fl.Field().String()
		if len(val) != 9 {
			return false
		}
		if !strings.HasPrefix(val, "4") {
			return false
		}
		for _, c := range val {
			if !unicode.IsDigit(c) {
				return false
			}
		}
		return true
	})

	return v
}

// ValidateStruct menjalankan seluruh aturan pada tag struct
func ValidateStruct(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var invalid *validator.InvalidValidationError
	if errors.As(err, &invalid) {
		return map[string]string{"_": "objek yang divalidasi tidak sah"}
	}

	var fieldErrors validator.ValidationErrors
	if !errors.As(err, &fieldErrors) {
		return map[string]string{"_": "validasi gagal"}
	}

	result := make(map[string]string, len(fieldErrors))
	for _, fe := range fieldErrors {
		if _, exists := result[fe.Field()]; !exists {
			result[fe.Field()] = messageFor(fe)
		}
	}
	return result
}

// messageFor menerjemahkan tag bahasa Inggris jadi pesan bahasa Indonesia
func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		if fe.Kind() == reflect.String {
			return "minimal " + fe.Param() + " karakter"
		}
		return "nilai minimal " + fe.Param()
	case "max":
		if fe.Kind() == reflect.String {
			return "maksimal " + fe.Param() + " karakter"
		}
		return "nilai maksimal " + fe.Param()
	case "alphanum":
		return "hanya boleh berisi huruf dan angka"
	case "nospace":
		return "tidak boleh mengandung spasi"
	case "username":
		return "hanya boleh huruf, angka, titik, dan garis bawah"
	case "strongpassword":
		return "password tidak memenuhi syarat (minimal 8 karakter)"
	case "nim":
		return "format NIM salah (harus 9 digit angka dan diawali dengan 4)"
	case "oneof":
		return "harus salah satu dari: " + strings.ReplaceAll(fe.Param(), " ", ", ")
	case "omitnil":
		return ""
	default:
		return "tidak memenuhi aturan " + fe.Tag()
	}
}
