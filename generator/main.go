package main

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	. "google.golang.org/protobuf/reflect/protoreflect"
)

func getFieldKindName(field *protogen.Field) (string, error) {
	var prefix = ""
	// TODO: support for optional fields
	//if(field.Desc.HasOptionalKeyword()) {
	prefix = "?"
	//}

	if field.Desc.IsMap() {
		return "", fmt.Errorf("Maps are not supported, field type: %s", field.Desc.Name)
	}

	if field.Desc.IsList() {
		prefix += "ArrayList("
	}

	switch field.Desc.Kind() {
	case Sint32Kind, Sfixed32Kind:
		prefix += "i32"
	case Uint32Kind, Fixed32Kind:
		prefix += "u32"
	case Sint64Kind, Sfixed64Kind:
		prefix += "i64"
	case Uint64Kind, Fixed64Kind:
		prefix += "u64"
	case BoolKind:
		prefix += "bool"
	case DoubleKind:
		prefix += "f64"
	case FloatKind:
		prefix += "f32"
	case StringKind: // TODO: validate if repeated strings and bytes are supported
		prefix += "ArrayList(u8)"
	case BytesKind:
		prefix += "ArrayList(u8)"
	case MessageKind:
		prefix += string(field.Message.Desc.Name())
	case EnumKind:
		prefix += string(field.Enum.Desc.Name())
	default:
		return "", fmt.Errorf("unmanaged field type in getFieldKindName %s", field.Desc.Kind())
	}

	if field.Desc.IsList() {
		prefix += ")"
	}

	return prefix, nil
}

func getFieldDescriptor(field *protogen.Field) (string, error) {
	if field.Desc.IsList() {
		switch field.Desc.Kind() {
		case StringKind, BytesKind:
			return fmt.Sprintf("fd(%d, .{ .List = .FixedInt })", field.Desc.Number()), nil
		case MessageKind:
			return fmt.Sprintf("fd(%d, .{ .List = .SubMessage })", field.Desc.Number()), nil
		case EnumKind:
			return fmt.Sprintf("fd(%d, .{ .List = .{ .Varint = .ZigZagOptimized } })", field.Desc.Number()), nil
		default:
			return "", fmt.Errorf("unmanaged field type in  getFieldDescriptor %s", field.Desc.Kind())
		}
	} else {
		switch field.Desc.Kind() {
		case Sfixed64Kind, Sfixed32Kind, Fixed32Kind, Fixed64Kind, DoubleKind, FloatKind:
			return fmt.Sprintf("fd(%d, .FixedInt)", field.Desc.Number()), nil
		case Sint32Kind, Sint64Kind:
			return fmt.Sprintf("fd(%d, .{ .Varint = .ZigZagOptimized })", field.Desc.Number()), nil
		case Uint32Kind, Uint64Kind, BoolKind:
			return fmt.Sprintf("fd(%d, .{ .Varint = .Simple })", field.Desc.Number()), nil
		case StringKind, BytesKind:
			return fmt.Sprintf("fd(%d, .{ .List = .FixedInt })", field.Desc.Number()), nil
		case MessageKind:
			return fmt.Sprintf("fd(%d, .{ .SubMessage = {} })", field.Desc.Number()), nil
		case EnumKind:
			return fmt.Sprintf("fd(%d, .{ .Varint = .ZigZagOptimized })", field.Desc.Number()), nil
		default:
			return "", fmt.Errorf("unmanaged field type in  getFieldDescriptor %s", field.Desc.Kind())
		}
	}
}

func generateFieldDescriptor(field *protogen.Field, g *protogen.GeneratedFile) error {
	if fieldDesc, err := getFieldDescriptor(field); err != nil {
		return err
	} else {
		g.P("        .", field.Desc.Name(), " = ", fieldDesc, ",")
	}
	return nil
}

func generateFieldDef(field *protogen.Field, g *protogen.GeneratedFile) error {
	if fieldKindName, err := getFieldKindName(field); err != nil {
		return err
	} else {
		g.P("    ", field.Desc.Name(), ": ", fieldKindName, ",")
	}
	return nil
}

func generateFile(p *protogen.Plugin, f *protogen.File) error {
	// Skip generating file if there is no message.
	if len(f.Messages) == 0 {
		return nil
	}
	filename := f.GeneratedFilenamePrefix + ".pb.zig"
	g := p.NewGeneratedFile(filename, "")
	g.P("// Code generated by protoc-gen-zig")
	g.P()
	g.P("const std = @import(\"std\");")
	g.P("const mem = std.mem;")
	g.P("const Allocator = mem.Allocator;")
	g.P("const ArrayList = std.ArrayList;")
	g.P()
	g.P("const protobuf = @import(\"protobuf\");")
	g.P("const FieldDescriptor = protobuf.FieldDescriptor;")
	g.P("const pb_decode = protobuf.pb_decode;")
	g.P("const pb_encode = protobuf.pb_encode;")
	g.P("const pb_deinit = protobuf.pb_deinit;")
	g.P("const pb_init = protobuf.pb_init;")
	g.P("const fd = protobuf.fd;")
	g.P()

	for _, m := range f.Enums {
		msgName := m.Desc.Name()

		g.P("pub const ", msgName, " = enum(i32) {") // TODO: type
		for _, f := range m.Values {
			g.P("    ", f.Desc.Name(), " = ", f.Desc.Number(), ",")
		}
		g.P("    _,")
		g.P("};")
		g.P("")
	}

	for _, m := range f.Messages {
		msgName := m.Desc.Name()

		g.P("pub const ", msgName, " = struct {")

		// field definitions
		for _, field := range m.Fields {
			if err := generateFieldDef(field, g); err != nil {
				return err
			}
		}
		g.P()

		// field descriptors
		g.P("    pub const _desc_table = .{")
		for _, field := range m.Fields {
			if err := generateFieldDescriptor(field, g); err != nil {
				return err
			}
		}
		g.P("    };")
		g.P()

		g.P("    pub fn encode(self: ", msgName, ", allocator: Allocator) ![]u8 {")
		g.P("        return pb_encode(self, allocator);")
		g.P("    }")
		g.P()

		g.P("    pub fn decode(input: []const u8, allocator: Allocator) !", msgName, " {")
		g.P("        return pb_decode(", msgName, ", input, allocator);")
		g.P("    }")
		g.P()

		g.P("    pub fn init(allocator: Allocator) ", msgName, " {")
		g.P("        return pb_init(", msgName, ", allocator);")
		g.P("    }")
		g.P()

		g.P("    pub fn deinit(self: ", msgName, ") void {")
		g.P("        return pb_deinit(self);")
		g.P("    }")

		g.P("};")
		g.P("")
	}

	return nil
}

func main() {
	protogen.Options{}.Run(func(plugin *protogen.Plugin) error {
		for _, file := range plugin.Files {
			if !file.Generate {
				continue
			}

			if err := generateFile(plugin, file); err != nil {
				return err
			}
		}

		return nil
	})
}
