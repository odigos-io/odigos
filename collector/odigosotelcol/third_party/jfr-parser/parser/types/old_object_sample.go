package types

// old_object_sample.go adds support for the JVM's native jdk.OldObjectSample
// event (the built-in leak profiler). Upstream grafana/jfr-parser only binds
// async-profiler's profiler.LiveObject; this binds the HotSpot event so the
// odigos memory profiler can read true leak samples (objectSize + allocation
// stack + age) from a JVM-native Flight Recorder. The Parse method mirrors the
// generated LiveObject parser: a generic field walker that captures the fields
// we bind and skips the rest (including the constant-pool object/root refs).

import (
	"fmt"
	"io"
	"unsafe"

	"github.com/grafana/jfr-parser/parser/types/def"
)

type BindOldObjectSample struct {
	Temp   OldObjectSample
	Fields []BindFieldOldObjectSample
}

type BindFieldOldObjectSample struct {
	Field         *def.Field
	uint64        *uint64
	StackTraceRef *StackTraceRef
}

func NewBindOldObjectSample(typ *def.Class, typeMap *def.TypeMap) *BindOldObjectSample {
	res := new(BindOldObjectSample)
	res.Fields = make([]BindFieldOldObjectSample, 0, len(typ.Fields))
	for i := 0; i < len(typ.Fields); i++ {
		switch typ.Fields[i].Name {
		case "stackTrace":
			if typ.Fields[i].Equals(&def.Field{Name: "stackTrace", Type: typeMap.T_STACK_TRACE, ConstantPool: true, Array: false}) {
				res.Fields = append(res.Fields, BindFieldOldObjectSample{Field: &typ.Fields[i], StackTraceRef: &res.Temp.StackTrace})
			} else {
				res.Fields = append(res.Fields, BindFieldOldObjectSample{Field: &typ.Fields[i]})
			}
		case "objectSize":
			if typ.Fields[i].Equals(&def.Field{Name: "objectSize", Type: typeMap.T_LONG, ConstantPool: false, Array: false}) {
				res.Fields = append(res.Fields, BindFieldOldObjectSample{Field: &typ.Fields[i], uint64: &res.Temp.ObjectSize})
			} else {
				res.Fields = append(res.Fields, BindFieldOldObjectSample{Field: &typ.Fields[i]})
			}
		case "objectAge":
			if typ.Fields[i].Equals(&def.Field{Name: "objectAge", Type: typeMap.T_LONG, ConstantPool: false, Array: false}) {
				res.Fields = append(res.Fields, BindFieldOldObjectSample{Field: &typ.Fields[i], uint64: &res.Temp.ObjectAge})
			} else {
				res.Fields = append(res.Fields, BindFieldOldObjectSample{Field: &typ.Fields[i]})
			}
		default:
			res.Fields = append(res.Fields, BindFieldOldObjectSample{Field: &typ.Fields[i]}) // skip unbound field
		}
	}
	return res
}

type OldObjectSample struct {
	StackTrace StackTraceRef
	ObjectSize uint64
	ObjectAge  uint64
}

func (this *OldObjectSample) Parse(data []byte, bind *BindOldObjectSample, typeMap *def.TypeMap) (pos int, err error) {
	var (
		v64_  uint64
		v32_  uint32
		v16_  uint16
		s_    string
		b_    byte
		shift = uint(0)
		l     = len(data)
	)
	_ = v64_
	_ = v32_
	_ = v16_
	_ = s_
	for bindFieldIndex := 0; bindFieldIndex < len(bind.Fields); bindFieldIndex++ {
		bindArraySize := 1
		if bind.Fields[bindFieldIndex].Field.Array {
			v32_ = uint32(0)
			for shift = uint(0); ; shift += 7 {
				if shift >= 32 {
					return 0, def.ErrIntOverflow
				}
				if pos >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b_ = data[pos]
				pos++
				v32_ |= uint32(b_&0x7F) << shift
				if b_ < 0x80 {
					break
				}
			}
			bindArraySize = int(v32_)
		}
		for bindArrayIndex := 0; bindArrayIndex < bindArraySize; bindArrayIndex++ {
			if bind.Fields[bindFieldIndex].Field.ConstantPool {
				v64_ = 0
				for shift = uint(0); shift <= 56; shift += 7 {
					if pos >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b_ = data[pos]
					pos++
					if shift == 56 {
						v64_ |= uint64(b_&0xFF) << shift
						break
					} else {
						v64_ |= uint64(b_&0x7F) << shift
						if b_ < 0x80 {
							break
						}
					}
				}
				switch bind.Fields[bindFieldIndex].Field.Type {
				case typeMap.T_STACK_TRACE:
					if bind.Fields[bindFieldIndex].StackTraceRef != nil {
						*bind.Fields[bindFieldIndex].StackTraceRef = StackTraceRef(v64_)
					}
				}
			} else {
				bindFieldTypeID := bind.Fields[bindFieldIndex].Field.Type
				switch bindFieldTypeID {
				case typeMap.T_STRING:
					s_ = ""
					if pos >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b_ = data[pos]
					pos++
					switch b_ {
					case 0:
						break
					case 1:
						break
					case 3:
						v32_ = uint32(0)
						for shift = uint(0); ; shift += 7 {
							if shift >= 32 {
								return 0, def.ErrIntOverflow
							}
							if pos >= l {
								return 0, io.ErrUnexpectedEOF
							}
							b_ = data[pos]
							pos++
							v32_ |= uint32(b_&0x7F) << shift
							if b_ < 0x80 {
								break
							}
						}
						if pos+int(v32_) > l {
							return 0, io.ErrUnexpectedEOF
						}
						bs := data[pos : pos+int(v32_)]
						s_ = *(*string)(unsafe.Pointer(&bs))
						pos += int(v32_)
					case 5:
						v32_ = uint32(0)
						for shift = uint(0); ; shift += 7 {
							if shift >= 32 {
								return 0, def.ErrIntOverflow
							}
							if pos >= l {
								return 0, io.ErrUnexpectedEOF
							}
							b_ = data[pos]
							pos++
							v32_ |= uint32(b_&0x7F) << shift
							if b_ < 0x80 {
								break
							}
						}
						if pos+int(v32_) > l {
							return 0, io.ErrUnexpectedEOF
						}
						bs := data[pos : pos+int(v32_)]
						bs, _ = typeMap.ISO8859_1Decoder.Bytes(bs)
						s_ = *(*string)(unsafe.Pointer(&bs))
						pos += int(v32_)
					case 4:
						v32_ = uint32(0)
						for shift = uint(0); ; shift += 7 {
							if shift >= 32 {
								return 0, def.ErrIntOverflow
							}
							if pos >= l {
								return 0, io.ErrUnexpectedEOF
							}
							b_ = data[pos]
							pos++
							v32_ |= uint32(b_&0x7F) << shift
							if b_ < 0x80 {
								break
							}
						}
						bl := int(v32_)
						buf := make([]rune, bl)
						for i := 0; i < bl; i++ {
							v32_ = uint32(0)
							for shift = uint(0); ; shift += 7 {
								if shift >= 32 {
									return 0, def.ErrIntOverflow
								}
								if pos >= l {
									return 0, io.ErrUnexpectedEOF
								}
								b_ = data[pos]
								pos++
								v32_ |= uint32(b_&0x7F) << shift
								if b_ < 0x80 {
									break
								}
							}
							buf[i] = rune(v32_)
						}
						s_ = string(buf)
					default:
						return 0, fmt.Errorf("unknown string type %d at %d", b_, pos)
					}
				case typeMap.T_INT:
					v32_ = uint32(0)
					for shift = uint(0); ; shift += 7 {
						if shift >= 32 {
							return 0, def.ErrIntOverflow
						}
						if pos >= l {
							return 0, io.ErrUnexpectedEOF
						}
						b_ = data[pos]
						pos++
						v32_ |= uint32(b_&0x7F) << shift
						if b_ < 0x80 {
							break
						}
					}
				case typeMap.T_LONG:
					v64_ = 0
					for shift = uint(0); shift <= 56; shift += 7 {
						if pos >= l {
							return 0, io.ErrUnexpectedEOF
						}
						b_ = data[pos]
						pos++
						if shift == 56 {
							v64_ |= uint64(b_&0xFF) << shift
							break
						} else {
							v64_ |= uint64(b_&0x7F) << shift
							if b_ < 0x80 {
								break
							}
						}
					}
					if bind.Fields[bindFieldIndex].uint64 != nil {
						*bind.Fields[bindFieldIndex].uint64 = v64_
					}
				case typeMap.T_SHORT:
					v16_ = uint16(0)
					for shift = uint(0); ; shift += 7 {
						if shift >= 16 {
							return 0, def.ErrIntOverflow
						}
						if pos >= l {
							return 0, io.ErrUnexpectedEOF
						}
						b_ = data[pos]
						pos++
						v16_ |= uint16(b_&0x7F) << shift
						if b_ < 0x80 {
							break
						}
					}
				case typeMap.T_BOOLEAN:
					if pos >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b_ = data[pos]
					pos++
				case typeMap.T_FLOAT:
					v32_ = uint32(0)
					for shift = uint(0); ; shift += 7 {
						if shift >= 32 {
							return 0, def.ErrIntOverflow
						}
						if pos >= l {
							return 0, io.ErrUnexpectedEOF
						}
						b_ = data[pos]
						pos++
						v32_ |= uint32(b_&0x7F) << shift
						if b_ < 0x80 {
							break
						}
					}
				default:
					bindFieldType := typeMap.IDMap[bind.Fields[bindFieldIndex].Field.Type]
					if bindFieldType == nil || len(bindFieldType.Fields) == 0 {
						return 0, fmt.Errorf("unknown type %d %+v", bind.Fields[bindFieldIndex].Field.Type, bindFieldType)
					}
					bindSkipObjects := 1
					if bind.Fields[bindFieldIndex].Field.Array {
						v32_ = uint32(0)
						for shift = uint(0); ; shift += 7 {
							if shift >= 32 {
								return 0, def.ErrIntOverflow
							}
							if pos >= l {
								return 0, io.ErrUnexpectedEOF
							}
							b_ = data[pos]
							pos++
							v32_ |= uint32(b_&0x7F) << shift
							if b_ < 0x80 {
								break
							}
						}
						bindSkipObjects = int(v32_)
					}
					for bindSkipObjectIndex := 0; bindSkipObjectIndex < bindSkipObjects; bindSkipObjectIndex++ {
						for bindskipFieldIndex := 0; bindskipFieldIndex < len(bindFieldType.Fields); bindskipFieldIndex++ {
							bindSkipFieldType := bindFieldType.Fields[bindskipFieldIndex].Type
							if bindFieldType.Fields[bindskipFieldIndex].ConstantPool {
								v32_ = uint32(0)
								for shift = uint(0); ; shift += 7 {
									if shift >= 32 {
										return 0, def.ErrIntOverflow
									}
									if pos >= l {
										return 0, io.ErrUnexpectedEOF
									}
									b_ = data[pos]
									pos++
									v32_ |= uint32(b_&0x7F) << shift
									if b_ < 0x80 {
										break
									}
								}
							} else if bindSkipFieldType == typeMap.T_STRING {
								s_ = ""
								if pos >= l {
									return 0, io.ErrUnexpectedEOF
								}
								b_ = data[pos]
								pos++
								switch b_ {
								case 0:
									break
								case 1:
									break
								case 3:
									v32_ = uint32(0)
									for shift = uint(0); ; shift += 7 {
										if shift >= 32 {
											return 0, def.ErrIntOverflow
										}
										if pos >= l {
											return 0, io.ErrUnexpectedEOF
										}
										b_ = data[pos]
										pos++
										v32_ |= uint32(b_&0x7F) << shift
										if b_ < 0x80 {
											break
										}
									}
									if pos+int(v32_) > l {
										return 0, io.ErrUnexpectedEOF
									}
									bs := data[pos : pos+int(v32_)]
									s_ = *(*string)(unsafe.Pointer(&bs))
									pos += int(v32_)
								case 5:
									v32_ = uint32(0)
									for shift = uint(0); ; shift += 7 {
										if shift >= 32 {
											return 0, def.ErrIntOverflow
										}
										if pos >= l {
											return 0, io.ErrUnexpectedEOF
										}
										b_ = data[pos]
										pos++
										v32_ |= uint32(b_&0x7F) << shift
										if b_ < 0x80 {
											break
										}
									}
									if pos+int(v32_) > l {
										return 0, io.ErrUnexpectedEOF
									}
									bs := data[pos : pos+int(v32_)]
									bs, _ = typeMap.ISO8859_1Decoder.Bytes(bs)
									s_ = *(*string)(unsafe.Pointer(&bs))
									pos += int(v32_)
								case 4:
									v32_ = uint32(0)
									for shift = uint(0); ; shift += 7 {
										if shift >= 32 {
											return 0, def.ErrIntOverflow
										}
										if pos >= l {
											return 0, io.ErrUnexpectedEOF
										}
										b_ = data[pos]
										pos++
										v32_ |= uint32(b_&0x7F) << shift
										if b_ < 0x80 {
											break
										}
									}
									bl := int(v32_)
									buf := make([]rune, bl)
									for i := 0; i < bl; i++ {
										v32_ = uint32(0)
										for shift = uint(0); ; shift += 7 {
											if shift >= 32 {
												return 0, def.ErrIntOverflow
											}
											if pos >= l {
												return 0, io.ErrUnexpectedEOF
											}
											b_ = data[pos]
											pos++
											v32_ |= uint32(b_&0x7F) << shift
											if b_ < 0x80 {
												break
											}
										}
										buf[i] = rune(v32_)
									}
									s_ = string(buf)
								default:
									return 0, fmt.Errorf("unknown string type %d at %d", b_, pos)
								}
							} else if bindSkipFieldType == typeMap.T_INT {
								v32_ = uint32(0)
								for shift = uint(0); ; shift += 7 {
									if shift >= 32 {
										return 0, def.ErrIntOverflow
									}
									if pos >= l {
										return 0, io.ErrUnexpectedEOF
									}
									b_ = data[pos]
									pos++
									v32_ |= uint32(b_&0x7F) << shift
									if b_ < 0x80 {
										break
									}
								}
							} else if bindSkipFieldType == typeMap.T_FLOAT {
								v32_ = uint32(0)
								for shift = uint(0); ; shift += 7 {
									if shift >= 32 {
										return 0, def.ErrIntOverflow
									}
									if pos >= l {
										return 0, io.ErrUnexpectedEOF
									}
									b_ = data[pos]
									pos++
									v32_ |= uint32(b_&0x7F) << shift
									if b_ < 0x80 {
										break
									}
								}
							} else if bindSkipFieldType == typeMap.T_LONG {
								v64_ = 0
								for shift = uint(0); shift <= 56; shift += 7 {
									if pos >= l {
										return 0, io.ErrUnexpectedEOF
									}
									b_ = data[pos]
									pos++
									if shift == 56 {
										v64_ |= uint64(b_&0xFF) << shift
										break
									} else {
										v64_ |= uint64(b_&0x7F) << shift
										if b_ < 0x80 {
											break
										}
									}
								}
							} else if bindSkipFieldType == typeMap.T_SHORT {
								v16_ = uint16(0)
								for shift = uint(0); ; shift += 7 {
									if shift >= 16 {
										return 0, def.ErrIntOverflow
									}
									if pos >= l {
										return 0, io.ErrUnexpectedEOF
									}
									b_ = data[pos]
									pos++
									v16_ |= uint16(b_&0x7F) << shift
									if b_ < 0x80 {
										break
									}
								}
							} else if bindSkipFieldType == typeMap.T_BOOLEAN {
								if pos >= l {
									return 0, io.ErrUnexpectedEOF
								}
								b_ = data[pos]
								pos++
							} else {
								return 0, fmt.Errorf("nested objects not implemented. ")
							}
						}
					}
				}
			}
		}
	}
	*this = bind.Temp
	return pos, nil
}
