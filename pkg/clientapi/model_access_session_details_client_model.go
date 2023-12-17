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

// checks if the AccessSessionDetailsClientModel type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &AccessSessionDetailsClientModel{}

// AccessSessionDetailsClientModel struct for AccessSessionDetailsClientModel
type AccessSessionDetailsClientModel struct {
	Credentials  NullableAccessSessionClientModelCredentials `json:"credentials,omitempty"`
	Instructions InstructionClientModel                      `json:"instructions"`
	Json         map[string]interface{}                      `json:"json,omitempty"`
	Cli          NullableString                              `json:"cli,omitempty"`
	Link         NullableAccessSessionDetailsClientModelLink `json:"link,omitempty"`
}

type _AccessSessionDetailsClientModel AccessSessionDetailsClientModel

// NewAccessSessionDetailsClientModel instantiates a new AccessSessionDetailsClientModel object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewAccessSessionDetailsClientModel(instructions InstructionClientModel) *AccessSessionDetailsClientModel {
	this := AccessSessionDetailsClientModel{}
	this.Instructions = instructions
	return &this
}

// NewAccessSessionDetailsClientModelWithDefaults instantiates a new AccessSessionDetailsClientModel object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewAccessSessionDetailsClientModelWithDefaults() *AccessSessionDetailsClientModel {
	this := AccessSessionDetailsClientModel{}
	return &this
}

// GetCredentials returns the Credentials field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *AccessSessionDetailsClientModel) GetCredentials() AccessSessionClientModelCredentials {
	if o == nil || IsNil(o.Credentials.Get()) {
		var ret AccessSessionClientModelCredentials
		return ret
	}
	return *o.Credentials.Get()
}

// GetCredentialsOk returns a tuple with the Credentials field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *AccessSessionDetailsClientModel) GetCredentialsOk() (*AccessSessionClientModelCredentials, bool) {
	if o == nil {
		return nil, false
	}
	return o.Credentials.Get(), o.Credentials.IsSet()
}

// HasCredentials returns a boolean if a field has been set.
func (o *AccessSessionDetailsClientModel) HasCredentials() bool {
	if o != nil && o.Credentials.IsSet() {
		return true
	}

	return false
}

// SetCredentials gets a reference to the given NullableAccessSessionClientModelCredentials and assigns it to the Credentials field.
func (o *AccessSessionDetailsClientModel) SetCredentials(v AccessSessionClientModelCredentials) {
	o.Credentials.Set(&v)
}

// SetCredentialsNil sets the value for Credentials to be an explicit nil
func (o *AccessSessionDetailsClientModel) SetCredentialsNil() {
	o.Credentials.Set(nil)
}

// UnsetCredentials ensures that no value is present for Credentials, not even an explicit nil
func (o *AccessSessionDetailsClientModel) UnsetCredentials() {
	o.Credentials.Unset()
}

// GetInstructions returns the Instructions field value
func (o *AccessSessionDetailsClientModel) GetInstructions() InstructionClientModel {
	if o == nil {
		var ret InstructionClientModel
		return ret
	}

	return o.Instructions
}

// GetInstructionsOk returns a tuple with the Instructions field value
// and a boolean to check if the value has been set.
func (o *AccessSessionDetailsClientModel) GetInstructionsOk() (*InstructionClientModel, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Instructions, true
}

// SetInstructions sets field value
func (o *AccessSessionDetailsClientModel) SetInstructions(v InstructionClientModel) {
	o.Instructions = v
}

// GetJson returns the Json field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *AccessSessionDetailsClientModel) GetJson() map[string]interface{} {
	if o == nil {
		var ret map[string]interface{}
		return ret
	}
	return o.Json
}

// GetJsonOk returns a tuple with the Json field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *AccessSessionDetailsClientModel) GetJsonOk() (map[string]interface{}, bool) {
	if o == nil || IsNil(o.Json) {
		return map[string]interface{}{}, false
	}
	return o.Json, true
}

// HasJson returns a boolean if a field has been set.
func (o *AccessSessionDetailsClientModel) HasJson() bool {
	if o != nil && IsNil(o.Json) {
		return true
	}

	return false
}

// SetJson gets a reference to the given map[string]interface{} and assigns it to the Json field.
func (o *AccessSessionDetailsClientModel) SetJson(v map[string]interface{}) {
	o.Json = v
}

// GetCli returns the Cli field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *AccessSessionDetailsClientModel) GetCli() string {
	if o == nil || IsNil(o.Cli.Get()) {
		var ret string
		return ret
	}
	return *o.Cli.Get()
}

// GetCliOk returns a tuple with the Cli field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *AccessSessionDetailsClientModel) GetCliOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Cli.Get(), o.Cli.IsSet()
}

// HasCli returns a boolean if a field has been set.
func (o *AccessSessionDetailsClientModel) HasCli() bool {
	if o != nil && o.Cli.IsSet() {
		return true
	}

	return false
}

// SetCli gets a reference to the given NullableString and assigns it to the Cli field.
func (o *AccessSessionDetailsClientModel) SetCli(v string) {
	o.Cli.Set(&v)
}

// SetCliNil sets the value for Cli to be an explicit nil
func (o *AccessSessionDetailsClientModel) SetCliNil() {
	o.Cli.Set(nil)
}

// UnsetCli ensures that no value is present for Cli, not even an explicit nil
func (o *AccessSessionDetailsClientModel) UnsetCli() {
	o.Cli.Unset()
}

// GetLink returns the Link field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *AccessSessionDetailsClientModel) GetLink() AccessSessionDetailsClientModelLink {
	if o == nil || IsNil(o.Link.Get()) {
		var ret AccessSessionDetailsClientModelLink
		return ret
	}
	return *o.Link.Get()
}

// GetLinkOk returns a tuple with the Link field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *AccessSessionDetailsClientModel) GetLinkOk() (*AccessSessionDetailsClientModelLink, bool) {
	if o == nil {
		return nil, false
	}
	return o.Link.Get(), o.Link.IsSet()
}

// HasLink returns a boolean if a field has been set.
func (o *AccessSessionDetailsClientModel) HasLink() bool {
	if o != nil && o.Link.IsSet() {
		return true
	}

	return false
}

// SetLink gets a reference to the given NullableAccessSessionDetailsClientModelLink and assigns it to the Link field.
func (o *AccessSessionDetailsClientModel) SetLink(v AccessSessionDetailsClientModelLink) {
	o.Link.Set(&v)
}

// SetLinkNil sets the value for Link to be an explicit nil
func (o *AccessSessionDetailsClientModel) SetLinkNil() {
	o.Link.Set(nil)
}

// UnsetLink ensures that no value is present for Link, not even an explicit nil
func (o *AccessSessionDetailsClientModel) UnsetLink() {
	o.Link.Unset()
}

func (o AccessSessionDetailsClientModel) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o AccessSessionDetailsClientModel) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if o.Credentials.IsSet() {
		toSerialize["credentials"] = o.Credentials.Get()
	}
	toSerialize["instructions"] = o.Instructions
	if o.Json != nil {
		toSerialize["json"] = o.Json
	}
	if o.Cli.IsSet() {
		toSerialize["cli"] = o.Cli.Get()
	}
	if o.Link.IsSet() {
		toSerialize["link"] = o.Link.Get()
	}
	return toSerialize, nil
}

func (o *AccessSessionDetailsClientModel) UnmarshalJSON(bytes []byte) (err error) {
	// This validates that all required properties are included in the JSON object
	// by unmarshalling the object into a generic map with string keys and checking
	// that every required field exists as a key in the generic map.
	requiredProperties := []string{
		"instructions",
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

	varAccessSessionDetailsClientModel := _AccessSessionDetailsClientModel{}

	err = json.Unmarshal(bytes, &varAccessSessionDetailsClientModel)

	if err != nil {
		return err
	}

	*o = AccessSessionDetailsClientModel(varAccessSessionDetailsClientModel)

	return err
}

type NullableAccessSessionDetailsClientModel struct {
	value *AccessSessionDetailsClientModel
	isSet bool
}

func (v NullableAccessSessionDetailsClientModel) Get() *AccessSessionDetailsClientModel {
	return v.value
}

func (v *NullableAccessSessionDetailsClientModel) Set(val *AccessSessionDetailsClientModel) {
	v.value = val
	v.isSet = true
}

func (v NullableAccessSessionDetailsClientModel) IsSet() bool {
	return v.isSet
}

func (v *NullableAccessSessionDetailsClientModel) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableAccessSessionDetailsClientModel(val *AccessSessionDetailsClientModel) *NullableAccessSessionDetailsClientModel {
	return &NullableAccessSessionDetailsClientModel{value: val, isSet: true}
}

func (v NullableAccessSessionDetailsClientModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableAccessSessionDetailsClientModel) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
