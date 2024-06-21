// Copyright 2019 Gin Core Team. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"encoding/hex"
	"fmt"
	"mime/multipart"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMappingBaseTypes(t *testing.T) {
	intPtr := func(i int) *int {
		return &i
	}
	for _, tt := range []struct {
		name   string
		value  any
		form   string
		expect any
	}{
		{"base type", struct{ F int }{}, "9", int(9)},
		{"base type", struct{ F int8 }{}, "9", int8(9)},
		{"base type", struct{ F int16 }{}, "9", int16(9)},
		{"base type", struct{ F int32 }{}, "9", int32(9)},
		{"base type", struct{ F int64 }{}, "9", int64(9)},
		{"base type", struct{ F uint }{}, "9", uint(9)},
		{"base type", struct{ F uint8 }{}, "9", uint8(9)},
		{"base type", struct{ F uint16 }{}, "9", uint16(9)},
		{"base type", struct{ F uint32 }{}, "9", uint32(9)},
		{"base type", struct{ F uint64 }{}, "9", uint64(9)},
		{"base type", struct{ F bool }{}, "True", true},
		{"base type", struct{ F float32 }{}, "9.1", float32(9.1)},
		{"base type", struct{ F float64 }{}, "9.1", float64(9.1)},
		{"base type", struct{ F string }{}, "test", string("test")},
		{"base type", struct{ F *int }{}, "9", intPtr(9)},

		// zero values
		{"zero value", struct{ F int }{}, "", int(0)},
		{"zero value", struct{ F uint }{}, "", uint(0)},
		{"zero value", struct{ F bool }{}, "", false},
		{"zero value", struct{ F float32 }{}, "", float32(0)},
		{"file value", struct{ F *multipart.FileHeader }{}, "", &multipart.FileHeader{}},
	} {
		tp := reflect.TypeOf(tt.value)
		testName := tt.name + ":" + tp.Field(0).Type.String()

		val := reflect.New(reflect.TypeOf(tt.value))
		val.Elem().Set(reflect.ValueOf(tt.value))

		field := val.Elem().Type().Field(0)

		_, err := mapping(val, emptyField, formSource{field.Name: {tt.form}}, "form")
		assert.NoError(t, err, testName)

		actual := val.Elem().Field(0).Interface()
		assert.Equal(t, tt.expect, actual, testName)
	}
}

func TestMappingDefault(t *testing.T) {
	var s struct {
		Int   int    `form:",default=9"`
		Slice []int  `form:",default=9"`
		Array [1]int `form:",default=9"`
	}
	err := mappingByPtr(&s, formSource{}, "form")
	assert.NoError(t, err)

	assert.Equal(t, 9, s.Int)
	assert.Equal(t, []int{9}, s.Slice)
	assert.Equal(t, [1]int{9}, s.Array)
}

func TestMappingSkipField(t *testing.T) {
	var s struct {
		A int
	}
	err := mappingByPtr(&s, formSource{}, "form")
	assert.NoError(t, err)

	assert.Equal(t, 0, s.A)
}

func TestMappingIgnoreField(t *testing.T) {
	var s struct {
		A int `form:"A"`
		B int `form:"-"`
	}
	err := mappingByPtr(&s, formSource{"A": {"9"}, "B": {"9"}}, "form")
	assert.NoError(t, err)

	assert.Equal(t, 9, s.A)
	assert.Equal(t, 0, s.B)
}

func TestMappingUnexportedField(t *testing.T) {
	var s struct {
		A int `form:"a"`
		b int `form:"b"`
	}
	err := mappingByPtr(&s, formSource{"a": {"9"}, "b": {"9"}}, "form")
	assert.NoError(t, err)

	assert.Equal(t, 9, s.A)
	assert.Equal(t, 0, s.b)
}

func TestMappingPrivateField(t *testing.T) {
	var s struct {
		f int `form:"field"`
	}
	err := mappingByPtr(&s, formSource{"field": {"6"}}, "form")
	assert.NoError(t, err)
	assert.Equal(t, 0, s.f)
}

func TestMappingUnknownFieldType(t *testing.T) {
	var s struct {
		U uintptr
	}

	err := mappingByPtr(&s, formSource{"U": {"unknown"}}, "form")
	assert.Error(t, err)
	assert.Equal(t, errUnknownType, err)
}

func TestMappingURI(t *testing.T) {
	var s struct {
		F int `uri:"field"`
	}
	err := mapURI(&s, map[string][]string{"field": {"6"}})
	assert.NoError(t, err)
	assert.Equal(t, 6, s.F)
}

func TestMappingForm(t *testing.T) {
	var s struct {
		F int `form:"field"`
	}
	err := mapForm(&s, map[string][]string{"field": {"6"}})
	assert.NoError(t, err)
	assert.Equal(t, 6, s.F)
}

func TestMapFormWithTag(t *testing.T) {
	var s struct {
		F int `externalTag:"field"`
	}
	err := MapFormWithTag(&s, map[string][]string{"field": {"6"}}, "externalTag")
	assert.NoError(t, err)
	assert.Equal(t, 6, s.F)
}

func TestMappingTime(t *testing.T) {
	var s struct {
		Time      time.Time
		LocalTime time.Time `time_format:"2006-01-02"`
		ZeroValue time.Time
		CSTTime   time.Time `time_format:"2006-01-02" time_location:"Asia/Shanghai"`
		UTCTime   time.Time `time_format:"2006-01-02" time_utc:"1"`
	}

	var err error
	time.Local, err = time.LoadLocation("Europe/Berlin")
	assert.NoError(t, err)

	err = mapForm(&s, map[string][]string{
		"Time":      {"2019-01-20T16:02:58Z"},
		"LocalTime": {"2019-01-20"},
		"ZeroValue": {},
		"CSTTime":   {"2019-01-20"},
		"UTCTime":   {"2019-01-20"},
	})
	assert.NoError(t, err)

	assert.Equal(t, "2019-01-20 16:02:58 +0000 UTC", s.Time.String())
	assert.Equal(t, "2019-01-20 00:00:00 +0100 CET", s.LocalTime.String())
	assert.Equal(t, "2019-01-19 23:00:00 +0000 UTC", s.LocalTime.UTC().String())
	assert.Equal(t, "0001-01-01 00:00:00 +0000 UTC", s.ZeroValue.String())
	assert.Equal(t, "2019-01-20 00:00:00 +0800 CST", s.CSTTime.String())
	assert.Equal(t, "2019-01-19 16:00:00 +0000 UTC", s.CSTTime.UTC().String())
	assert.Equal(t, "2019-01-20 00:00:00 +0000 UTC", s.UTCTime.String())

	// wrong location
	var wrongLoc struct {
		Time time.Time `time_location:"wrong"`
	}
	err = mapForm(&wrongLoc, map[string][]string{"Time": {"2019-01-20T16:02:58Z"}})
	assert.Error(t, err)

	// wrong time value
	var wrongTime struct {
		Time time.Time
	}
	err = mapForm(&wrongTime, map[string][]string{"Time": {"wrong"}})
	assert.Error(t, err)
}

func TestMappingTimeDuration(t *testing.T) {
	var s struct {
		D time.Duration
	}

	// ok
	err := mappingByPtr(&s, formSource{"D": {"5s"}}, "form")
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Second, s.D)

	// error
	err = mappingByPtr(&s, formSource{"D": {"wrong"}}, "form")
	assert.Error(t, err)
}

func TestMappingAny(t *testing.T) {
	type sT struct {
		Value   any
		PValue  *any
		PPValue **any
		DV      any  `form:"dv,default=aa"`
		DV2     any  `form:"dv2,default=aa2"`
		DV3     *any `form:"dv3,default=aaa3"`
	}

	noNil := func(st *sT) bool {
		if st.PValue == nil || st.PPValue == nil || *(st.PPValue) == nil || st.DV3 == nil {
			return false
		}
		return true
	}

	var s sT
	// ok
	err := mappingByPtr(&s, formSource{"Value": {"1"}, "PValue": {"p1"}, "PPValue": {"pp1"}, "dv2": {"aaa2"}}, "form")
	assert.NoError(t, err)
	assert.True(t, noNil(&s))
	assert.Equal(t, "1", s.Value)
	assert.Equal(t, "p1", *(s.PValue))
	assert.Equal(t, "pp1", *(*(s.PPValue)))
	assert.Equal(t, "aa", s.DV)
	assert.Equal(t, "aaa2", s.DV2)
	assert.Equal(t, "aaa3", *(s.DV3))

	var s2 sT
	// ok
	err = mappingByPtr(&s2, formSource{"Value": {"1", "a2"}, "PValue": {"p1", "2.0"}, "PPValue": {"pp1", "2.00"}, "dv3": {"3", "33"}}, "form")
	assert.NoError(t, err)
	assert.True(t, noNil(&s2))
	assert.Equal(t, []string{"1", "a2"}, s2.Value)
	assert.Equal(t, []string{"p1", "2.0"}, *(s2.PValue))
	assert.Equal(t, []string{"pp1", "2.00"}, *(*(s2.PPValue)))
	assert.Equal(t, []string{"3", "33"}, *(s2.DV3))
}

func TestMappingSliceArrayAny(t *testing.T) {
	var s struct {
		Values    []any
		PValues   *[]any
		PPValues  **[]any
		AValues   [2]any
		PAValues  *[2]any
		PPAValues **[2]any
	}

	noNil := func() bool {
		if s.PValues == nil || s.PPValues == nil || *(s.PPValues) == nil {
			return false
		}
		if s.PAValues == nil || s.PPAValues == nil || *(s.PPAValues) == nil {
			return false
		}
		return true
	}

	// ok
	err := mappingByPtr(&s,
		formSource{
			"Values": {"1"}, "PValues": {"p1"}, "PPValues": {"pp1"},
			"AValues": {"a1", "a2"}, "PAValues": {"pa1", "pa2"}, "PPAValues": {"ppa1", "ppa2"},
		}, "form")
	assert.NoError(t, err)
	assert.True(t, noNil())
	assert.Equal(t, []any{"1"}, s.Values)
	assert.Equal(t, []any{"p1"}, *(s.PValues))
	assert.Equal(t, []any{"pp1"}, *(*(s.PPValues)))

	assert.Equal(t, [2]any{"a1", "a2"}, s.AValues)
	assert.Equal(t, [2]any{"pa1", "pa2"}, *(s.PAValues))
	assert.Equal(t, [2]any{"ppa1", "ppa2"}, *(*(s.PPAValues)))

	// error - not enough vals
	err = mappingByPtr(&s,
		formSource{
			"Values": {"1"}, "PValues": {"p1"}, "PPValues": {"pp1"},
			"AValues": {"a1", "a2"}, "PAValues": {"pa1"}, "PPAValues": {"ppa1", "ppa2"},
		}, "form")
	assert.Error(t, err)
}

func TestMappingSlice(t *testing.T) {
	var s struct {
		Slice []int `form:"slice,default=9"`
	}

	// default value
	err := mappingByPtr(&s, formSource{}, "form")
	assert.NoError(t, err)
	assert.Equal(t, []int{9}, s.Slice)

	// ok
	err = mappingByPtr(&s, formSource{"slice": {"3", "4"}}, "form")
	assert.NoError(t, err)
	assert.Equal(t, []int{3, 4}, s.Slice)

	// error
	err = mappingByPtr(&s, formSource{"slice": {"wrong"}}, "form")
	assert.Error(t, err)
}

func TestMappingArray(t *testing.T) {
	var s struct {
		Array [2]int `form:"array,default=9"`
	}

	// wrong default
	err := mappingByPtr(&s, formSource{}, "form")
	assert.Error(t, err)

	// ok
	err = mappingByPtr(&s, formSource{"array": {"3", "4"}}, "form")
	assert.NoError(t, err)
	assert.Equal(t, [2]int{3, 4}, s.Array)

	// error - not enough vals
	err = mappingByPtr(&s, formSource{"array": {"3"}}, "form")
	assert.Error(t, err)

	// error - wrong value
	err = mappingByPtr(&s, formSource{"array": {"wrong"}}, "form")
	assert.Error(t, err)
}

func TestMappingStructField(t *testing.T) {
	var s struct {
		J struct {
			I int
		}
	}

	err := mappingByPtr(&s, formSource{"J": {`{"I": 9}`}}, "form")
	assert.NoError(t, err)
	assert.Equal(t, 9, s.J.I)
}

func TestMappingPtrField(t *testing.T) {
	type ptrStruct struct {
		Key int64 `json:"key"`
	}

	type ptrRequest struct {
		Items []*ptrStruct `json:"items" form:"items"`
	}

	var err error

	// With 0 items.
	var req0 ptrRequest
	err = mappingByPtr(&req0, formSource{}, "form")
	assert.NoError(t, err)
	assert.Empty(t, req0.Items)

	// With 1 item.
	var req1 ptrRequest
	err = mappingByPtr(&req1, formSource{"items": {`{"key": 1}`}}, "form")
	assert.NoError(t, err)
	assert.Len(t, req1.Items, 1)
	assert.EqualValues(t, 1, req1.Items[0].Key)

	// With 2 items.
	var req2 ptrRequest
	err = mappingByPtr(&req2, formSource{"items": {`{"key": 1}`, `{"key": 2}`}}, "form")
	assert.NoError(t, err)
	assert.Len(t, req2.Items, 2)
	assert.EqualValues(t, 1, req2.Items[0].Key)
	assert.EqualValues(t, 2, req2.Items[1].Key)
}

func TestMappingMapField(t *testing.T) {
	var s struct {
		M map[string]int
	}

	err := mappingByPtr(&s, formSource{"M": {`{"one": 1}`}}, "form")
	assert.NoError(t, err)
	assert.Equal(t, map[string]int{"one": 1}, s.M)
}

func TestMappingIgnoredCircularRef(t *testing.T) {
	type S struct {
		S *S `form:"-"`
	}
	var s S

	err := mappingByPtr(&s, formSource{}, "form")
	assert.NoError(t, err)
}

type customUnmarshalParamHex int

func (f *customUnmarshalParamHex) UnmarshalParam(param string) error {
	v, err := strconv.ParseInt(param, 16, 64)
	if err != nil {
		return err
	}
	*f = customUnmarshalParamHex(v)
	return nil
}

func TestMappingCustomUnmarshalParamHexWithFormTag(t *testing.T) {
	var s struct {
		Foo customUnmarshalParamHex `form:"foo"`
	}
	err := mappingByPtr(&s, formSource{"foo": {`f5`}}, "form")
	assert.NoError(t, err)

	assert.EqualValues(t, 245, s.Foo)
}

func TestMappingCustomUnmarshalParamHexWithURITag(t *testing.T) {
	var s struct {
		Foo customUnmarshalParamHex `uri:"foo"`
	}
	err := mappingByPtr(&s, formSource{"foo": {`f5`}}, "uri")
	assert.NoError(t, err)

	assert.EqualValues(t, 245, s.Foo)
}

type customUnmarshalParamType struct {
	Protocol string
	Path     string
	Name     string
}

func (f *customUnmarshalParamType) UnmarshalParam(param string) error {
	parts := strings.Split(param, ":")
	if len(parts) != 3 {
		return fmt.Errorf("invalid format")
	}
	f.Protocol = parts[0]
	f.Path = parts[1]
	f.Name = parts[2]
	return nil
}

func TestMappingCustomStructTypeWithFormTag(t *testing.T) {
	var s struct {
		FileData customUnmarshalParamType `form:"data"`
	}
	err := mappingByPtr(&s, formSource{"data": {`file:/foo:happiness`}}, "form")
	assert.NoError(t, err)

	assert.EqualValues(t, "file", s.FileData.Protocol)
	assert.EqualValues(t, "/foo", s.FileData.Path)
	assert.EqualValues(t, "happiness", s.FileData.Name)
}

func TestMappingCustomStructTypeWithURITag(t *testing.T) {
	var s struct {
		FileData customUnmarshalParamType `uri:"data"`
	}
	err := mappingByPtr(&s, formSource{"data": {`file:/foo:happiness`}}, "uri")
	assert.NoError(t, err)

	assert.EqualValues(t, "file", s.FileData.Protocol)
	assert.EqualValues(t, "/foo", s.FileData.Path)
	assert.EqualValues(t, "happiness", s.FileData.Name)
}

func TestMappingCustomPointerStructTypeWithFormTag(t *testing.T) {
	var s struct {
		FileData *customUnmarshalParamType `form:"data"`
	}
	err := mappingByPtr(&s, formSource{"data": {`file:/foo:happiness`}}, "form")
	assert.NoError(t, err)

	assert.EqualValues(t, "file", s.FileData.Protocol)
	assert.EqualValues(t, "/foo", s.FileData.Path)
	assert.EqualValues(t, "happiness", s.FileData.Name)
}

func TestMappingCustomPointerStructTypeWithURITag(t *testing.T) {
	var s struct {
		FileData *customUnmarshalParamType `uri:"data"`
	}
	err := mappingByPtr(&s, formSource{"data": {`file:/foo:happiness`}}, "uri")
	assert.NoError(t, err)

	assert.EqualValues(t, "file", s.FileData.Protocol)
	assert.EqualValues(t, "/foo", s.FileData.Path)
	assert.EqualValues(t, "happiness", s.FileData.Name)
}

type customPath []string

func (p *customPath) UnmarshalParam(param string) error {
	elems := strings.Split(param, "/")
	n := len(elems)
	if n < 2 {
		return fmt.Errorf("invalid format")
	}

	*p = elems
	return nil
}

func TestMappingCustomSliceUri(t *testing.T) {
	var s struct {
		FileData customPath `uri:"path"`
	}
	err := mappingByPtr(&s, formSource{"path": {`bar/foo`}}, "uri")
	assert.NoError(t, err)

	assert.EqualValues(t, "bar", s.FileData[0])
	assert.EqualValues(t, "foo", s.FileData[1])
}

func TestMappingCustomSliceForm(t *testing.T) {
	var s struct {
		FileData customPath `form:"path"`
	}
	err := mappingByPtr(&s, formSource{"path": {`bar/foo`}}, "form")
	assert.NoError(t, err)

	assert.EqualValues(t, "bar", s.FileData[0])
	assert.EqualValues(t, "foo", s.FileData[1])
}

type objectID [12]byte

func (o *objectID) UnmarshalParam(param string) error {
	oid, err := convertTo(param)
	if err != nil {
		return err
	}

	*o = oid
	return nil
}

func convertTo(s string) (objectID, error) {
	var nilObjectID objectID
	if len(s) != 24 {
		return nilObjectID, fmt.Errorf("invalid format")
	}

	var oid [12]byte
	_, err := hex.Decode(oid[:], []byte(s))
	if err != nil {
		return nilObjectID, err
	}

	return oid, nil
}

func TestMappingCustomArrayUri(t *testing.T) {
	var s struct {
		FileData objectID `uri:"id"`
	}
	val := `664a062ac74a8ad104e0e80f`
	err := mappingByPtr(&s, formSource{"id": {val}}, "uri")
	assert.NoError(t, err)

	expected, _ := convertTo(val)
	assert.EqualValues(t, expected, s.FileData)
}

func TestMappingCustomArrayForm(t *testing.T) {
	var s struct {
		FileData objectID `form:"id"`
	}
	val := `664a062ac74a8ad104e0e80f`
	err := mappingByPtr(&s, formSource{"id": {val}}, "form")
	assert.NoError(t, err)

	expected, _ := convertTo(val)
	assert.EqualValues(t, expected, s.FileData)
}
