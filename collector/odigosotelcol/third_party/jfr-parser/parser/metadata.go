package parser

import (
	"fmt"

	"github.com/grafana/jfr-parser/parser/types/def"
	"golang.org/x/text/encoding/charmap"
)

func (p *Parser) readMeta(pos int) error {
	p.TypeMap.IDMap = make(map[def.TypeID]*def.Class, 43+5)
	p.TypeMap.NameMap = make(map[string]*def.Class, 43+5)
	p.TypeMap.ISO8859_1Decoder = charmap.ISO8859_1.NewDecoder()

	if err := p.seek(pos); err != nil {
		return err
	}
	sz, err := p.varInt()
	if err != nil {
		return err
	}
	p.metaSize = sz
	_, err = p.varInt()
	if err != nil {
		return err
	}
	_, err = p.varLong()
	if err != nil {
		return err
	}
	_, err = p.varLong()
	if err != nil {
		return err
	}
	_, err = p.varLong()
	if err != nil {
		return err
	}
	nstr, err := p.varInt()
	if err != nil {
		return err
	}
	strings := make([]string, nstr)
	for i := 0; i < int(nstr); i++ {
		strings[i], err = p.string()
		if err != nil {
			return err
		}
	}

	e, err := p.readElement(strings, false)
	if err != nil {
		return err
	}
	if e.name != "root" {
		return fmt.Errorf("expected root element, got %s", e.name)
	}
	for i := 0; i < e.childCount; i++ {
		meta, err := p.readElement(strings, false)
		if err != nil {
			return err
		}
		//fmt.Println(meta.name)
		switch meta.name {
		case "metadata":
			for j := 0; j < meta.childCount; j++ {
				classElement, err := p.readElement(strings, true)

				if err != nil {
					return err
				}
				cls, err := def.NewClass(classElement.attr, classElement.childCount)
				if err != nil {
					return err
				}

				for k := 0; k < classElement.childCount; k++ {
					field, err := p.readElement(strings, true)
					if err != nil {
						return err
					}
					if field.name == "field" {
						f, err := def.NewField(field.attr)
						if err != nil {
							return err
						}
						cls.Fields = append(cls.Fields, f)
					}
					for l := 0; l < field.childCount; l++ {
						_, err := p.readElement(strings, false)
						if err != nil {
							return err
						}
					}

				}
				p.TypeMap.IDMap[cls.ID] = cls
				p.TypeMap.NameMap[cls.Name] = cls

			}
		case "region":
			break
		default:
			return fmt.Errorf("unexpected element %s", meta.name)
		}
	}
	if err := p.checkTypes(); err != nil {
		return err
	}
	return nil
}

func (p *Parser) readElement(strings []string, needAttributes bool) (element, error) {
	iname, err := p.varInt()
	if err != nil {
		return element{}, err
	}
	if iname < 0 || int(iname) >= len(strings) {
		return element{}, def.ErrIntOverflow
	}
	name := strings[iname]
	attributeCount, err := p.varInt()
	if err != nil {
		return element{}, err
	}
	var attributes map[string]string
	if needAttributes {
		attributes = make(map[string]string, attributeCount)
	}
	for i := 0; i < int(attributeCount); i++ {
		attributeName, err := p.varInt()
		if err != nil {
			return element{}, err
		}
		if attributeName < 0 || int(attributeName) >= len(strings) {
			return element{}, def.ErrIntOverflow
		}
		attributeValue, err := p.varInt()
		if err != nil {
			return element{}, err
		}
		if attributeValue < 0 || int(attributeValue) >= len(strings) {
			return element{}, def.ErrIntOverflow
		}
		if needAttributes {
			attributes[strings[attributeName]] = strings[attributeValue]
		} else {
			//fmt.Printf("                              >>> skipping attribute %s=%s\n", strings[attributeName], strings[attributeValue])
		}
	}

	childCount, err := p.varInt()
	if err != nil {
		return element{}, err
	}
	return element{
		name:       name,
		attr:       attributes,
		childCount: int(childCount),
	}, nil

}

type element struct {
	name       string
	attr       map[string]string
	childCount int
}
