package principal

import (
	"errors"
	"reflect"
)

type (
	// Id represents the identity of a subject. i.e. user
	Id string

	// Role represents a certain level of authorization. i.e. operator
	Role string

	// Group organizes users having common capabilities. i.e. admin
	Group string

	// User identifies a username or account name. i.e. test1
	User string
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