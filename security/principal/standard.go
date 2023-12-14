package principal

import (
	"errors"
	"reflect"
)

type (
	// Id represents the identity of a subject.
	// i.e. user
	Id string

	// User identifies a username or account name.
	// i.e. test1
	User string

	// Email is used to hold the email address of the subject.
	// i.e. johm.doe@domain.com
	Email string

	// Role represents a certain level of authorization.
	// i.e. operator
	Role string

	// Group organizes users having common capabilities.
	// i.e. admin
	Group string

	// Entitlement refers to the rights and privileges granted to a user or a group.
	// i.e. createWidget
	Entitlement string
)

//goland:noinspection GoMixedReceiverTypes
func (i Id) Name() string {
	return string(i)
}

//goland:noinspection GoMixedReceiverTypes
func (i *Id) InitWithTag(tag reflect.StructTag) error {
	if name, ok := tag.Lookup("name"); ok {
		if name == "" {
			return errors.New("id name is required")
		}
		*i = Id(name)
	}
	return nil
}

//goland:noinspection GoMixedReceiverTypes
func (u User) Name() string {
	return string(u)
}

//goland:noinspection GoMixedReceiverTypes
func (u *User) InitWithTag(tag reflect.StructTag) error {
	if name, ok := tag.Lookup("name"); ok {
		if name == "" {
			return errors.New("user name is required")
		}
		*u = User(name)
	}
	return nil
}

//goland:noinspection GoMixedReceiverTypes
func (e Email) Name() string {
	return string(e)
}

//goland:noinspection GoMixedReceiverTypes
func (e *Email) InitWithTag(tag reflect.StructTag) error {
	if name, ok := tag.Lookup("name"); ok {
		if name == "" {
			return errors.New("email name is required")
		}
		*e = Email(name)
	}
	return nil
}

//goland:noinspection GoMixedReceiverTypes
func (r Role) Name() string {
	return string(r)
}

//goland:noinspection GoMixedReceiverTypes
func (r *Role) InitWithTag(tag reflect.StructTag) error {
	if name, ok := tag.Lookup("name"); ok {
		if name == "" {
			return errors.New("role name is required")
		}
		*r = Role(name)
	}
	return nil
}

//goland:noinspection GoMixedReceiverTypes
func (g Group) Name() string {
	return string(g)
}

//goland:noinspection GoMixedReceiverTypes
func (g *Group) InitWithTag(tag reflect.StructTag) error {
	if name, ok := tag.Lookup("name"); ok {
		if name == "" {
			return errors.New("group name is required")
		}
		*g = Group(name)
	}
	return nil
}

//goland:noinspection GoMixedReceiverTypes
func (e Entitlement) Name() string {
	return string(e)
}

//goland:noinspection GoMixedReceiverTypes
func (e *Entitlement) InitWithTag(tag reflect.StructTag) error {
	if name, ok := tag.Lookup("name"); ok {
		if name == "" {
			return errors.New("entitlement name is required")
		}
		*e = Entitlement(name)
	}
	return nil
}
