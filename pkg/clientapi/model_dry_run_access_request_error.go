/*
Apono

No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)

API version: 1.0.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package clientapi

import (
	"encoding/json"
	"fmt"
)

// checks if the DryRunAccessRequestError type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &DryRunAccessRequestError{}

// DryRunAccessRequestError struct for DryRunAccessRequestError
type DryRunAccessRequestError struct {
	Code    string                 `json:"code"`
	Field   string                 `json:"field"`
	Message NullableString         `json:"message,omitempty"`
	Details map[string]interface{} `json:"details"`
}

type _DryRunAccessRequestError DryRunAccessRequestError

// NewDryRunAccessRequestError instantiates a new DryRunAccessRequestError object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewDryRunAccessRequestError(code string, field string, details map[string]interface{}) *DryRunAccessRequestError {
	this := DryRunAccessRequestError{}
	this.Code = code
	this.Field = field
	this.Details = details
	return &this
}

// NewDryRunAccessRequestErrorWithDefaults instantiates a new DryRunAccessRequestError object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewDryRunAccessRequestErrorWithDefaults() *DryRunAccessRequestError {
	this := DryRunAccessRequestError{}
	return &this
}

// GetCode returns the Code field value
func (o *DryRunAccessRequestError) GetCode() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Code
}

// GetCodeOk returns a tuple with the Code field value
// and a boolean to check if the value has been set.
func (o *DryRunAccessRequestError) GetCodeOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Code, true
}

// SetCode sets field value
func (o *DryRunAccessRequestError) SetCode(v string) {
	o.Code = v
}

// GetField returns the Field field value
func (o *DryRunAccessRequestError) GetField() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Field
}

// GetFieldOk returns a tuple with the Field field value
// and a boolean to check if the value has been set.
func (o *DryRunAccessRequestError) GetFieldOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Field, true
}

// SetField sets field value
func (o *DryRunAccessRequestError) SetField(v string) {
	o.Field = v
}

// GetMessage returns the Message field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *DryRunAccessRequestError) GetMessage() string {
	if o == nil || IsNil(o.Message.Get()) {
		var ret string
		return ret
	}
	return *o.Message.Get()
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *DryRunAccessRequestError) GetMessageOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Message.Get(), o.Message.IsSet()
}

// HasMessage returns a boolean if a field has been set.
func (o *DryRunAccessRequestError) HasMessage() bool {
	if o != nil && o.Message.IsSet() {
		return true
	}

	return false
}

// SetMessage gets a reference to the given NullableString and assigns it to the Message field.
func (o *DryRunAccessRequestError) SetMessage(v string) {
	o.Message.Set(&v)
}

// SetMessageNil sets the value for Message to be an explicit nil
func (o *DryRunAccessRequestError) SetMessageNil() {
	o.Message.Set(nil)
}

// UnsetMessage ensures that no value is present for Message, not even an explicit nil
func (o *DryRunAccessRequestError) UnsetMessage() {
	o.Message.Unset()
}

// GetDetails returns the Details field value
func (o *DryRunAccessRequestError) GetDetails() map[string]interface{} {
	if o == nil {
		var ret map[string]interface{}
		return ret
	}

	return o.Details
}

// GetDetailsOk returns a tuple with the Details field value
// and a boolean to check if the value has been set.
func (o *DryRunAccessRequestError) GetDetailsOk() (map[string]interface{}, bool) {
	if o == nil {
		return map[string]interface{}{}, false
	}
	return o.Details, true
}

// SetDetails sets field value
func (o *DryRunAccessRequestError) SetDetails(v map[string]interface{}) {
	o.Details = v
}

func (o DryRunAccessRequestError) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o DryRunAccessRequestError) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["code"] = o.Code
	toSerialize["field"] = o.Field
	if o.Message.IsSet() {
		toSerialize["message"] = o.Message.Get()
	}
	toSerialize["details"] = o.Details
	return toSerialize, nil
}

func (o *DryRunAccessRequestError) UnmarshalJSON(bytes []byte) (err error) {
	// This validates that all required properties are included in the JSON object
	// by unmarshalling the object into a generic map with string keys and checking
	// that every required field exists as a key in the generic map.
	requiredProperties := []string{
		"code",
		"field",
		"details",
	}

	allProperties := make(map[string]interface{})

	err = json.Unmarshal(bytes, &allProperties)

	if err != nil {
		return err
	}

	for _, requiredProperty := range requiredProperties {
		if _, exists := allProperties[requiredProperty]; !exists {
			return fmt.Errorf("no value given for required property %v", requiredProperty)
		}
	}

	varDryRunAccessRequestError := _DryRunAccessRequestError{}

	err = json.Unmarshal(bytes, &varDryRunAccessRequestError)

	if err != nil {
		return err
	}

	*o = DryRunAccessRequestError(varDryRunAccessRequestError)

	return err
}

type NullableDryRunAccessRequestError struct {
	value *DryRunAccessRequestError
	isSet bool
}

func (v NullableDryRunAccessRequestError) Get() *DryRunAccessRequestError {
	return v.value
}

func (v *NullableDryRunAccessRequestError) Set(val *DryRunAccessRequestError) {
	v.value = val
	v.isSet = true
}

func (v NullableDryRunAccessRequestError) IsSet() bool {
	return v.isSet
}

func (v *NullableDryRunAccessRequestError) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableDryRunAccessRequestError(val *DryRunAccessRequestError) *NullableDryRunAccessRequestError {
	return &NullableDryRunAccessRequestError{value: val, isSet: true}
}

func (v NullableDryRunAccessRequestError) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableDryRunAccessRequestError) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
