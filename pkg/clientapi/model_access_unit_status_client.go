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

// AccessUnitStatusClient the model 'AccessUnitStatusClient'
type AccessUnitStatusClient string

// List of AccessUnitStatusClient
const (
	ACCESSUNITSTATUSCLIENT_GRANTED AccessUnitStatusClient = "Granted"
	ACCESSUNITSTATUSCLIENT_FAILURE AccessUnitStatusClient = "Failure"
	ACCESSUNITSTATUSCLIENT_PENDING AccessUnitStatusClient = "Pending"
	ACCESSUNITSTATUSCLIENT_REVOKED AccessUnitStatusClient = "Revoked"
)

// All allowed values of AccessUnitStatusClient enum
var AllowedAccessUnitStatusClientEnumValues = []AccessUnitStatusClient{
	"Granted",
	"Failure",
	"Pending",
	"Revoked",
}

func (v *AccessUnitStatusClient) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	enumTypeValue := AccessUnitStatusClient(value)
	for _, existing := range AllowedAccessUnitStatusClientEnumValues {
		if existing == enumTypeValue {
			*v = enumTypeValue
			return nil
		}
	}

	return fmt.Errorf("%+v is not a valid AccessUnitStatusClient", value)
}

// NewAccessUnitStatusClientFromValue returns a pointer to a valid AccessUnitStatusClient
// for the value passed as argument, or an error if the value passed is not allowed by the enum
func NewAccessUnitStatusClientFromValue(v string) (*AccessUnitStatusClient, error) {
	ev := AccessUnitStatusClient(v)
	if ev.IsValid() {
		return &ev, nil
	} else {
		return nil, fmt.Errorf("invalid value '%v' for AccessUnitStatusClient: valid values are %v", v, AllowedAccessUnitStatusClientEnumValues)
	}
}

// IsValid return true if the value is valid for the enum, false otherwise
func (v AccessUnitStatusClient) IsValid() bool {
	for _, existing := range AllowedAccessUnitStatusClientEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to AccessUnitStatusClient value
func (v AccessUnitStatusClient) Ptr() *AccessUnitStatusClient {
	return &v
}

type NullableAccessUnitStatusClient struct {
	value *AccessUnitStatusClient
	isSet bool
}

func (v NullableAccessUnitStatusClient) Get() *AccessUnitStatusClient {
	return v.value
}

func (v *NullableAccessUnitStatusClient) Set(val *AccessUnitStatusClient) {
	v.value = val
	v.isSet = true
}

func (v NullableAccessUnitStatusClient) IsSet() bool {
	return v.isSet
}

func (v *NullableAccessUnitStatusClient) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableAccessUnitStatusClient(val *AccessUnitStatusClient) *NullableAccessUnitStatusClient {
	return &NullableAccessUnitStatusClient{value: val, isSet: true}
}

func (v NullableAccessUnitStatusClient) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableAccessUnitStatusClient) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
