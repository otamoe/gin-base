package server

import (
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/globalsign/mgo/bson"
	mgoModel "github.com/otamoe/mgo-model"
	validator "gopkg.in/go-playground/validator.v9"
)

type (
	ginValidator struct {
	}
	modelValidator struct {
	}
)

var _ binding.StructValidator = &ginValidator{}

func (v *ginValidator) ValidateStruct(data interface{}) (err error) {
	value := reflect.ValueOf(data)
	valueType := value.Kind()

	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	if valueType == reflect.Struct {
		if err = Validate.Struct(data); err != nil {
			return
		}
	}
	return
}

func (v *ginValidator) Engine() interface{} {
	return Validate
}

func (v *modelValidator) ValidateDocument(document mgoModel.DocumentInterface) (err error) {
	if err = Validate.Struct(document); err != nil {
		return
	}
	return
}

var (
	Validate       *validator.Validate
	dnsNameRegex   = regexp.MustCompile("^(?:[0-9a-zA-Z]+(?:\\-[0-9a-zA-Z]+)*)$")
	urlSchemeRegex = regexp.MustCompile("^(?:[0-9a-zA-Z]+(?:[.\\-][0-9a-zA-Z]+)*)$")
	localeRegex    = regexp.MustCompile("^[a-z]{2}(\\-[A-Z][a-z]{2,3})?(\\-[A-Z]{2})?$")
)

func validationObjectId(fl validator.FieldLevel) bool {
	fieldInterface := fl.Field().Interface()
	switch fieldInterface.(type) {
	case bson.ObjectId:
		objectid := fieldInterface.(bson.ObjectId)
		return objectid.Valid()
	case string:
		id := fieldInterface.(string)
		return bson.IsObjectIdHex(id)
	}
	return false
}

func validationDnsName(fl validator.FieldLevel) bool {
	return dnsNameRegex.MatchString(fl.Field().String())
}

func validationLocale(fl validator.FieldLevel) bool {
	return localeRegex.MatchString(fl.Field().String())
}

func validationPhone(fl validator.FieldLevel) bool {
	return false
}

func validationLower(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	val := field.String()
	return strings.ToLower(val) == val
}

func validationUpper(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	val := field.String()
	return strings.ToUpper(val) == val
}

func validationTrim(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	val := field.String()
	if fl.Param() == "" {
		return strings.TrimSpace(val) == val
	} else {
		return strings.Trim(val, fl.Param()) == val
	}
}

func validationEmailBlacklist(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	arr := strings.Split(field.String(), "@")
	if len(arr) != 2 {
		return false
	}
	name := strings.ToLower(arr[0])
	if strings.ContainsAny(name, ".+_#%&<>$") {
		return false
	}

	domain := strings.ToLower(arr[1])
	if !strings.ContainsAny(domain, ".") {
		return false
	}

	return true
}

func validationFuncLower(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	field.SetString(strings.ToLower(field.String()))
	return true
}

func validationFuncUpper(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	field.SetString(strings.ToUpper(field.String()))
	return true
}

func validationFuncTrim(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	if fl.Param() == "" {
		field.SetString(strings.TrimSpace(field.String()))
	} else {
		field.SetString(strings.Trim(field.String(), fl.Param()))
	}
	return true
}

func validationTimezone(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	value := field.String()
	if strings.ToLower(value) == "local" {
		return false
	}
	if _, err := time.LoadLocation(value); err != nil {
		return false
	}
	return true
}

func validationURLScheme(fl validator.FieldLevel) bool {

	// -javascript
	// -vbscript
	// -data

	field := fl.Field()
	param := strings.Split(fl.Param(), " ")
	allow := []string{}
	reject := []string{}
	for _, value := range param {
		value = strings.TrimSpace(strings.ToLower(value))
		if value == "" {
			continue
		}
		if value[0] == '-' {
			reject = append(reject, value[1:])
		} else {
			allow = append(allow, value)
		}
	}
	sort.Strings(allow)
	sort.Strings(reject)

	switch field.Kind() {
	case reflect.String:
		u, err := url.ParseRequestURI(field.String())
		if err != nil || u.Scheme == "" {
			return false
		}

		// 必须是 URLSchemeRegex
		if !urlSchemeRegex.MatchString(u.Scheme) {
			return false
		}

		// 小写
		scheme := strings.TrimSpace(strings.ToLower(u.Scheme))

		// 查找到返回 false
		if len(reject) != 0 {
			if i := sort.SearchStrings(reject, scheme); i != len(reject) && reject[i] == scheme {
				return false
			}
		}

		// 没找到返回 false
		if len(reject) == 0 || len(allow) != 0 {
			if i := sort.SearchStrings(allow, scheme); i == len(allow) || allow[i] != scheme {
				return false
			}
		}

		return true
	}
	return false
}

func init() {
	Validate = validator.New()
	Validate.SetTagName("binding")
	Validate.RegisterValidation("objectid", validationObjectId)
	Validate.RegisterValidation("dnsname", validationDnsName)
	Validate.RegisterValidation("timezone", validationTimezone)
	Validate.RegisterValidation("locale", validationLocale)
	Validate.RegisterValidation("phone", validationPhone)
	Validate.RegisterValidation("lower", validationLower)
	Validate.RegisterValidation("upper", validationUpper)
	Validate.RegisterValidation("trim", validationTrim)
	Validate.RegisterValidation("url_scheme", validationURLScheme)
	Validate.RegisterValidation("email_blacklist", validationEmailBlacklist)
	Validate.RegisterValidation("func-lower", validationFuncLower)
	Validate.RegisterValidation("func-upper", validationFuncUpper)
	Validate.RegisterValidation("func-trim", validationFuncTrim)
	binding.Validator = new(ginValidator)

	mgoModel.Validator = new(modelValidator)
}
