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

// checks if the UserSessionClientModel type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &UserSessionClientModel{}

// UserSessionClientModel struct for UserSessionClientModel
type UserSessionClientModel struct {
	User    UserClientModel    `json:"user"`
	Account AccountClientModel `json:"account"`
}

type _UserSessionClientModel UserSessionClientModel

// NewUserSessionClientModel instantiates a new UserSessionClientModel object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewUserSessionClientModel(user UserClientModel, account AccountClientModel) *UserSessionClientModel {
	this := UserSessionClientModel{}
	this.User = user
	this.Account = account
	return &this
}

// NewUserSessionClientModelWithDefaults instantiates a new UserSessionClientModel object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewUserSessionClientModelWithDefaults() *UserSessionClientModel {
	this := UserSessionClientModel{}
	return &this
}

// GetUser returns the User field value
func (o *UserSessionClientModel) GetUser() UserClientModel {
	if o == nil {
		var ret UserClientModel
		return ret
	}

	return o.User
}

// GetUserOk returns a tuple with the User field value
// and a boolean to check if the value has been set.
func (o *UserSessionClientModel) GetUserOk() (*UserClientModel, bool) {
	if o == nil {
		return nil, false
	}
	return &o.User, true
}

// SetUser sets field value
func (o *UserSessionClientModel) SetUser(v UserClientModel) {
	o.User = v
}

// GetAccount returns the Account field value
func (o *UserSessionClientModel) GetAccount() AccountClientModel {
	if o == nil {
		var ret AccountClientModel
		return ret
	}

	return o.Account
}

// GetAccountOk returns a tuple with the Account field value
// and a boolean to check if the value has been set.
func (o *UserSessionClientModel) GetAccountOk() (*AccountClientModel, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Account, true
}

// SetAccount sets field value
func (o *UserSessionClientModel) SetAccount(v AccountClientModel) {
	o.Account = v
}

func (o UserSessionClientModel) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o UserSessionClientModel) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["user"] = o.User
	toSerialize["account"] = o.Account
	return toSerialize, nil
}

func (o *UserSessionClientModel) UnmarshalJSON(bytes []byte) (err error) {
	// This validates that all required properties are included in the JSON object
	// by unmarshalling the object into a generic map with string keys and checking
	// that every required field exists as a key in the generic map.
	requiredProperties := []string{
		"user",
		"account",
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

	varUserSessionClientModel := _UserSessionClientModel{}

	err = json.Unmarshal(bytes, &varUserSessionClientModel)

	if err != nil {
		return err
	}

	*o = UserSessionClientModel(varUserSessionClientModel)

	return err
}

type NullableUserSessionClientModel struct {
	value *UserSessionClientModel
	isSet bool
}

func (v NullableUserSessionClientModel) Get() *UserSessionClientModel {
	return v.value
}

func (v *NullableUserSessionClientModel) Set(val *UserSessionClientModel) {
	v.value = val
	v.isSet = true
}

func (v NullableUserSessionClientModel) IsSet() bool {
	return v.isSet
}

func (v *NullableUserSessionClientModel) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableUserSessionClientModel(val *UserSessionClientModel) *NullableUserSessionClientModel {
	return &NullableUserSessionClientModel{value: val, isSet: true}
}

func (v NullableUserSessionClientModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableUserSessionClientModel) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}