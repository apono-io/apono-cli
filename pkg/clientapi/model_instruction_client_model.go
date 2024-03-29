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

// checks if the InstructionClientModel type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &InstructionClientModel{}

// InstructionClientModel struct for InstructionClientModel
type InstructionClientModel struct {
	Plain    string `json:"plain"`
	Markdown string `json:"markdown"`
}

type _InstructionClientModel InstructionClientModel

// NewInstructionClientModel instantiates a new InstructionClientModel object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewInstructionClientModel(plain string, markdown string) *InstructionClientModel {
	this := InstructionClientModel{}
	this.Plain = plain
	this.Markdown = markdown
	return &this
}

// NewInstructionClientModelWithDefaults instantiates a new InstructionClientModel object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewInstructionClientModelWithDefaults() *InstructionClientModel {
	this := InstructionClientModel{}
	return &this
}

// GetPlain returns the Plain field value
func (o *InstructionClientModel) GetPlain() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Plain
}

// GetPlainOk returns a tuple with the Plain field value
// and a boolean to check if the value has been set.
func (o *InstructionClientModel) GetPlainOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Plain, true
}

// SetPlain sets field value
func (o *InstructionClientModel) SetPlain(v string) {
	o.Plain = v
}

// GetMarkdown returns the Markdown field value
func (o *InstructionClientModel) GetMarkdown() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Markdown
}

// GetMarkdownOk returns a tuple with the Markdown field value
// and a boolean to check if the value has been set.
func (o *InstructionClientModel) GetMarkdownOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Markdown, true
}

// SetMarkdown sets field value
func (o *InstructionClientModel) SetMarkdown(v string) {
	o.Markdown = v
}

func (o InstructionClientModel) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o InstructionClientModel) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["plain"] = o.Plain
	toSerialize["markdown"] = o.Markdown
	return toSerialize, nil
}

func (o *InstructionClientModel) UnmarshalJSON(bytes []byte) (err error) {
	// This validates that all required properties are included in the JSON object
	// by unmarshalling the object into a generic map with string keys and checking
	// that every required field exists as a key in the generic map.
	requiredProperties := []string{
		"plain",
		"markdown",
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

	varInstructionClientModel := _InstructionClientModel{}

	err = json.Unmarshal(bytes, &varInstructionClientModel)

	if err != nil {
		return err
	}

	*o = InstructionClientModel(varInstructionClientModel)

	return err
}

type NullableInstructionClientModel struct {
	value *InstructionClientModel
	isSet bool
}

func (v NullableInstructionClientModel) Get() *InstructionClientModel {
	return v.value
}

func (v *NullableInstructionClientModel) Set(val *InstructionClientModel) {
	v.value = val
	v.isSet = true
}

func (v NullableInstructionClientModel) IsSet() bool {
	return v.isSet
}

func (v *NullableInstructionClientModel) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableInstructionClientModel(val *InstructionClientModel) *NullableInstructionClientModel {
	return &NullableInstructionClientModel{value: val, isSet: true}
}

func (v NullableInstructionClientModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableInstructionClientModel) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
