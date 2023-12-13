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

// AccessStatus the model 'AccessStatus'
type AccessStatus string

// List of AccessStatus
const (
	ACCESSSTATUS_PENDING  AccessStatus = "PENDING"
	ACCESSSTATUS_APPROVED AccessStatus = "APPROVED"
	ACCESSSTATUS_REJECTED AccessStatus = "REJECTED"
	ACCESSSTATUS_GRANTED  AccessStatus = "GRANTED"
	ACCESSSTATUS_REVOKING AccessStatus = "REVOKING"
	ACCESSSTATUS_EXPIRED  AccessStatus = "EXPIRED"
	ACCESSSTATUS_FAILED   AccessStatus = "FAILED"
)

// All allowed values of AccessStatus enum
var AllowedAccessStatusEnumValues = []AccessStatus{
	"PENDING",
	"APPROVED",
	"REJECTED",
	"GRANTED",
	"REVOKING",
	"EXPIRED",
	"FAILED",
}

func (v *AccessStatus) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	enumTypeValue := AccessStatus(value)
	for _, existing := range AllowedAccessStatusEnumValues {
		if existing == enumTypeValue {
			*v = enumTypeValue
			return nil
		}
	}

	return fmt.Errorf("%+v is not a valid AccessStatus", value)
}

// NewAccessStatusFromValue returns a pointer to a valid AccessStatus
// for the value passed as argument, or an error if the value passed is not allowed by the enum
func NewAccessStatusFromValue(v string) (*AccessStatus, error) {
	ev := AccessStatus(v)
	if ev.IsValid() {
		return &ev, nil
	} else {
		return nil, fmt.Errorf("invalid value '%v' for AccessStatus: valid values are %v", v, AllowedAccessStatusEnumValues)
	}
}

// IsValid return true if the value is valid for the enum, false otherwise
func (v AccessStatus) IsValid() bool {
	for _, existing := range AllowedAccessStatusEnumValues {
		if existing == v {
			return true
		}
	}
	return false
}

// Ptr returns reference to AccessStatus value
func (v AccessStatus) Ptr() *AccessStatus {
	return &v
}

type NullableAccessStatus struct {
	value *AccessStatus
	isSet bool
}

func (v NullableAccessStatus) Get() *AccessStatus {
	return v.value
}

func (v *NullableAccessStatus) Set(val *AccessStatus) {
	v.value = val
	v.isSet = true
}

func (v NullableAccessStatus) IsSet() bool {
	return v.isSet
}

func (v *NullableAccessStatus) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableAccessStatus(val *AccessStatus) *NullableAccessStatus {
	return &NullableAccessStatus{value: val, isSet: true}
}

func (v NullableAccessStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableAccessStatus) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
