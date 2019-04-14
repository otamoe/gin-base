package errors

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	validator9 "gopkg.in/go-playground/validator.v9"
)

type (
	Errors struct {
		Errors     []*Error               `json:"errors"`
		StatusCode int                    `json:"status_code,omitempty"`
		Maps       map[string]interface{} `json:"-"`
	}

	Error struct {
		Err        error                  `json:"-"`
		Message    string                 `json:"message,omitempty"`
		Name       string                 `json:"name,omitempty"`
		Type       string                 `json:"type,omitempty"`
		Path       string                 `json:"path,omitempty"`
		Value      interface{}            `json:"value,omitempty"`
		StatusCode int                    `json:"status_code,omitempty"`
		Params     map[string]interface{} `json:"params,omitempty"`
		Maps       map[string]interface{} `json:"-"`
	}
	Callback func(errors *Errors)
)

var CONTEXT_CALLBACK = "GIN.SERVER.ERRORS.CALLBACK"

func (b *Errors) JSON() map[string]interface{} {
	json := map[string]interface{}{
		"errors_text": b.String(),
		"status_code": b.StatusCode,
		"errors":      b.Errors,
	}
	for _, e := range b.Errors {
		for k, v := range e.Maps {
			json[k] = v
		}
	}
	for k, v := range b.Maps {
		json[k] = v
	}
	return json
}

func (b *Errors) String() string {
	var errorsText []string
	for _, e := range b.Errors {
		errorsText = append(errorsText, e.Error())
	}
	return strings.Join(errorsText, "\n")
}

func (b *Errors) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.JSON())
}

func (b *Errors) addStatusCode(statusCode int) {
	if statusCode > b.StatusCode || (b.StatusCode == http.StatusNotFound && statusCode >= http.StatusBadRequest) {
		b.StatusCode = statusCode
	}
}
func (b *Errors) setMeta(meta interface{}) {
	if meta == nil {
		return
	}
	metaV := reflect.ValueOf(meta)
	switch metaV.Kind() {
	case reflect.Map:
		if b.Maps == nil {
			b.Maps = map[string]interface{}{}
		}
		for _, key := range metaV.MapKeys() {
			b.Maps[key.String()] = metaV.MapIndex(key).Interface()
		}
	}
}

func (e *Error) JSON() map[string]interface{} {
	json := map[string]interface{}{}
	json["message"] = e.Error()
	if e.Name != "" {
		json["name"] = e.Name
	}

	if e.Type != "" {
		json["type"] = e.Type
	}

	if e.Path != "" {
		json["path"] = e.Path
	}

	if e.Value != nil {
		json["value"] = e.Value
	}

	if e.StatusCode != 0 {
		json["status_code"] = e.StatusCode
	}

	if e.Params != nil {
		for k, v := range e.Params {
			json[k] = v
		}
	}
	return json
}

func (e *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.JSON())
}

func (e Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}

func (e *Error) Clone() *Error {
	e2 := *e
	e3 := e2
	newE := &e3
	if newE.Params != nil {
		params := newE.Params
		newE.Params = map[string]interface{}{}
		for k, v := range params {
			newE.Params[k] = v
		}
	}
	if newE.Maps != nil {
		maps := newE.Maps
		newE.Maps = map[string]interface{}{}
		for k, v := range maps {
			newE.Maps[k] = v
		}
	}
	return newE
}

func toNameUnderline(name string) string {
	// 驼峰转 下划线
	path := []byte{}
	nameUp := 0
	nameBytes := []byte(name)
	for _, val := range nameBytes {
		if val >= 65 && val <= 90 {
			if nameUp == 0 && len(path) != 0 {
				val2 := path[len(path)-1]
				if val2 >= 48 && val2 <= 57 || val2 >= 97 && val2 <= 122 {
					path = append(path, 95)
				}
			}
			path = append(path, val+32)
			nameUp++
		} else {
			if nameUp > 1 {
				index := len(path) - 1
				val2 := path[index]
				path[index] = 95
				path = append(path, val2)
			}
			nameUp = 0
			path = append(path, val)
		}
	}
	return string(path)
}

func getValue(value interface{}) interface{} {
	if val, ok := value.(fmt.Stringer); ok {
		return getValueSubstr(val.String())
	}
	switch reflect.TypeOf(value).Kind() {
	case reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128:
		return value
	case reflect.String:
		valueString, _ := value.(string)
		return getValueSubstr(valueString)
	default:
		return fmt.Sprintf("%+v", value)
	}
}

func getValueSubstr(value string) string {
	if utf8.ValidString(value) {
		runes := []rune(value)
		if len(runes) > 64 {
			value = string(runes[0:61]) + "..."
		}
	} else if len(value) > 64 {
		value = value[0:61] + "..."
	}
	return value
}

func Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			// 恢复线程
			if err := recover(); err != nil {
				stack := stack(3)
				switch err.(type) {
				case string:
					ctx.Error(errors.New(err.(string))).SetMeta(string(stack))
				case error:
					ctx.Error(err.(error)).SetMeta(string(stack))
				default:
					ctx.Error(errors.New(fmt.Sprintf("%+v", err))).SetMeta(string(stack))
				}
			}

			if len(ctx.Errors) == 0 {
				return
			}

			errors := &Errors{}
			errors.StatusCode = ctx.Writer.Status()

			for _, val := range ctx.Errors {
				switch val.Err.(type) {
				case *Error:
					e := val.Err.(*Error)
					errors.Errors = append(errors.Errors, e)
					errors.addStatusCode(e.StatusCode)
					errors.setMeta(val.Meta)
				case validator9.ValidationErrors:
					validationErrors := val.Err.(validator9.ValidationErrors)
					for _, fieldError := range validationErrors {
						value := fieldError.Value()
						tag := fieldError.Tag()
						params := gin.H{}
						switch tag {
						case "required",
							"alpha",
							"alphanum",
							"numeric",
							"hexadecimal",
							"hexcolor",
							"rgb",
							"rgba",
							"hsl",
							"hsla",
							"email",
							"url",
							"uri",
							"base64",
							"isbn",
							"isbn10",
							"isbn13",
							"uuid",
							"uuid3",
							"uuid4",
							"uuid5",
							"ascii",
							"asciiprint",
							"multibyte",
							"datauri",
							"latitude",
							"longitude",
							"ssn",
							"ip",
							"ipv4",
							"ipv6",
							"cidr",
							"cidrv4",
							"cidrv6",
							"tcp_addr",
							"tcp4_addr",
							"tcp6_addr",
							"udp_addr",
							"udp4_addr",
							"udp6_addr",
							"ip_addr",
							"ip4_addr",
							"ip6_addr",
							"unix_addr",
							"mac",
							"iscolor",

							"objectid",
							"dnsname":
						case "max",
							"min",
							"len":
							if param, e := strconv.Atoi(fieldError.Param()); e == nil {
								params[tag] = param
							} else {
								params[tag] = 0
							}
						case "eq",
							"ne",
							"gt",
							"gte",
							"lt",
							"lte":
							if reflect.TypeOf(value).Kind() == reflect.String {
								params[tag] = fieldError.Param()
							} else if param, e := strconv.Atoi(fieldError.Param()); e == nil {
								params[tag] = param
							} else {
								params[tag] = fieldError.Param()
							}
						default:
							params[tag] = fieldError.Param()
						}
						path := fieldError.Namespace()
						if index := strings.Index(path, "."); index != -1 {
							path = path[index+1:]
						}
						errors.Errors = append(errors.Errors, &Error{
							Message:    fieldError.Translate(nil),
							Name:       "validation",
							Type:       tag,
							Path:       toNameUnderline(path),
							Value:      getValue(value),
							StatusCode: http.StatusBadRequest,
							Params:     params,
						})
					}
					errors.addStatusCode(http.StatusBadRequest)
					errors.setMeta(val.Meta)
				default:
					errors.Errors = append(errors.Errors, &Error{
						Err: val.Err,
					})
					errors.setMeta(val.Meta)
				}
			}

			if errors.StatusCode < http.StatusMultipleChoices {
				errors.StatusCode = http.StatusInternalServerError
			}

			// callback
			if val, ok := ctx.Get(CONTEXT_CALLBACK); ok && val != nil {
				if call, ok := val.(Callback); ok {
					call(errors)
				}
			}

			ctx.AbortWithStatusJSON(errors.StatusCode, errors)
		}()
		ctx.Next()
	}
}

var (
	dunno     = []byte("???")
	centerDot = []byte("·")
	dot       = []byte(".")
	slash     = []byte("/")
)

// stack returns a nicely formatted stack frame, skipping skip frames.
func stack(skip int) []byte {
	buf := new(bytes.Buffer) // the returned data
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ { // Skip the expected number of frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Print this much at least.  If we can't find the source, it won't show.
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

// source returns a space-trimmed slice of the n'th line.
func source(lines [][]byte, n int) []byte {
	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//	runtime/debug.*T·ptrmethod
	// and want
	//	*T.ptrmethod
	// Also the package path might contains dot (e.g. code.google.com/...),
	// so first eliminate the path prefix
	if lastslash := bytes.LastIndex(name, slash); lastslash >= 0 {
		name = name[lastslash+1:]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}
