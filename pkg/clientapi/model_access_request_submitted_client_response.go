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

// checks if the AccessRequestSubmittedClientResponse type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &AccessRequestSubmittedClientResponse{}

// AccessRequestSubmittedClientResponse struct for AccessRequestSubmittedClientResponse
type AccessRequestSubmittedClientResponse struct {
	RequestIds []string       `json:"request_ids"`
	Message    NullableString `json:"message,omitempty"`
}

type _AccessRequestSubmittedClientResponse AccessRequestSubmittedClientResponse

// NewAccessRequestSubmittedClientResponse instantiates a new AccessRequestSubmittedClientResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewAccessRequestSubmittedClientResponse(requestIds []string) *AccessRequestSubmittedClientResponse {
	this := AccessRequestSubmittedClientResponse{}
	this.RequestIds = requestIds
	return &this
}

// NewAccessRequestSubmittedClientResponseWithDefaults instantiates a new AccessRequestSubmittedClientResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewAccessRequestSubmittedClientResponseWithDefaults() *AccessRequestSubmittedClientResponse {
	this := AccessRequestSubmittedClientResponse{}
	return &this
}

// GetRequestIds returns the RequestIds field value
func (o *AccessRequestSubmittedClientResponse) GetRequestIds() []string {
	if o == nil {
		var ret []string
		return ret
	}

	return o.RequestIds
}

// GetRequestIdsOk returns a tuple with the RequestIds field value
// and a boolean to check if the value has been set.
func (o *AccessRequestSubmittedClientResponse) GetRequestIdsOk() ([]string, bool) {
	if o == nil {
		return nil, false
	}
	return o.RequestIds, true
}

// SetRequestIds sets field value
func (o *AccessRequestSubmittedClientResponse) SetRequestIds(v []string) {
	o.RequestIds = v
}

// GetMessage returns the Message field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *AccessRequestSubmittedClientResponse) GetMessage() string {
	if o == nil || IsNil(o.Message.Get()) {
		var ret string
		return ret
	}
	return *o.Message.Get()
}

// GetMessageOk returns a tuple with the Message field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *AccessRequestSubmittedClientResponse) GetMessageOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Message.Get(), o.Message.IsSet()
}

// HasMessage returns a boolean if a field has been set.
func (o *AccessRequestSubmittedClientResponse) HasMessage() bool {
	if o != nil && o.Message.IsSet() {
		return true
	}

	return false
}

// SetMessage gets a reference to the given NullableString and assigns it to the Message field.
func (o *AccessRequestSubmittedClientResponse) SetMessage(v string) {
	o.Message.Set(&v)
}

// SetMessageNil sets the value for Message to be an explicit nil
func (o *AccessRequestSubmittedClientResponse) SetMessageNil() {
	o.Message.Set(nil)
}

// UnsetMessage ensures that no value is present for Message, not even an explicit nil
func (o *AccessRequestSubmittedClientResponse) UnsetMessage() {
	o.Message.Unset()
}

func (o AccessRequestSubmittedClientResponse) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o AccessRequestSubmittedClientResponse) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["request_ids"] = o.RequestIds
	if o.Message.IsSet() {
		toSerialize["message"] = o.Message.Get()
	}
	return toSerialize, nil
}

func (o *AccessRequestSubmittedClientResponse) UnmarshalJSON(bytes []byte) (err error) {
	// This validates that all required properties are included in the JSON object
	// by unmarshalling the object into a generic map with string keys and checking
	// that every required field exists as a key in the generic map.
	requiredProperties := []string{
		"request_ids",
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

	varAccessRequestSubmittedClientResponse := _AccessRequestSubmittedClientResponse{}

	err = json.Unmarshal(bytes, &varAccessRequestSubmittedClientResponse)

	if err != nil {
		return err
	}

	*o = AccessRequestSubmittedClientResponse(varAccessRequestSubmittedClientResponse)

	return err
}

type NullableAccessRequestSubmittedClientResponse struct {
	value *AccessRequestSubmittedClientResponse
	isSet bool
}

func (v NullableAccessRequestSubmittedClientResponse) Get() *AccessRequestSubmittedClientResponse {
	return v.value
}

func (v *NullableAccessRequestSubmittedClientResponse) Set(val *AccessRequestSubmittedClientResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableAccessRequestSubmittedClientResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableAccessRequestSubmittedClientResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableAccessRequestSubmittedClientResponse(val *AccessRequestSubmittedClientResponse) *NullableAccessRequestSubmittedClientResponse {
	return &NullableAccessRequestSubmittedClientResponse{value: val, isSet: true}
}

func (v NullableAccessRequestSubmittedClientResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableAccessRequestSubmittedClientResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
