package def

import (
	"fmt"
	"strconv"
)

var ErrIntOverflow = fmt.Errorf("int overflow")
var ErrNameEmpty = fmt.Errorf("class/field name is empty")

type Class struct {
	Name   string
	ID     TypeID
	Fields []Field
}

func NewClass(attrs map[string]string, childCount int) (*Class, error) {
	id, err := strconv.Atoi(attrs["id"])
	if err != nil {
		return nil, err
	}
	name := attrs["name"]
	if name == "" {
		return nil, ErrNameEmpty
	}
	return &Class{
		Name:   name,
		ID:     TypeID(id),
		Fields: make([]Field, 0, childCount),
	}, nil
}

func (c *Class) String() string {
	if c == nil {
		return "class{nil}"
	}
	return fmt.Sprintf("class{name: %s, id: %d, fields: %+v}", c.Name, c.ID, c.Fields)
}

func (c *Class) TrimLastField(fieldName string) []Field {
	if len(c.Fields) > 0 && c.Fields[len(c.Fields)-1].Name == fieldName {
		return c.Fields[:len(c.Fields)-1]
	} else {
		return c.Fields
	}
}

func (c *Class) Field(name string) *Field {
	for i := range c.Fields {
		if c.Fields[i].Name == name {
			return &c.Fields[i]
		}
	}
	return nil
}

type Field struct {
	Name         string
	Type         TypeID
	ConstantPool bool
	Array        bool
}

func (f *Field) Equals(other *Field) bool {
	return f.Name == other.Name &&
		f.Type == other.Type &&
		f.ConstantPool == other.ConstantPool &&
		f.Array == other.Array
}

func (f *Field) String() string {
	return fmt.Sprintf("field{name: %s, typ: %d, constantPool: %t}", f.Name, f.Type, f.ConstantPool)
}

func NewField(attrs map[string]string) (Field, error) {
	cls := attrs["class"]
	typ, err := strconv.Atoi(cls)
	if err != nil {
		return Field{}, err
	}
	name := attrs["name"]
	if name == "" {
		return Field{}, ErrNameEmpty
	}
	dimen := attrs["dimension"]
	array := false
	if dimen != "" {
		if dimen == "1" {
			array = true
		} else {
			return Field{}, fmt.Errorf("unsupported dimension %s", dimen)
		}
	}

	return Field{
		Name:         name,
		Type:         TypeID(typ),
		ConstantPool: attrs["constantPool"] == "true",
		Array:        array,
	}, nil
}
