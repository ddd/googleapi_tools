package parser

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func GenerateProtoFile(fd protoreflect.FileDescriptor) string {
	var sb strings.Builder

	// Write syntax
	sb.WriteString(fmt.Sprintf("syntax = \"%s\";\n\n", fd.Syntax()))

	// Write package
	sb.WriteString(fmt.Sprintf("package %s;\n\n", fd.Package()))

	// Write imports
	for i := 0; i < fd.Imports().Len(); i++ {
		imp := fd.Imports().Get(i)
		sb.WriteString(fmt.Sprintf("import \"%s\";\n", imp.Path()))
	}
	if fd.Imports().Len() > 0 {
		sb.WriteString("\n")
	}

	// Write file-level enums
	for i := 0; i < fd.Enums().Len(); i++ {
		enum := fd.Enums().Get(i)
		generateEnum(&sb, enum, 0)
		sb.WriteString("\n")
	}

	// Write messages
	for i := 0; i < fd.Messages().Len(); i++ {
		msg := fd.Messages().Get(i)
		generateMessage(&sb, msg, 0)

		// Add a newline between root-level messages, but not after the last one
		if i < fd.Messages().Len()-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func generateEnum(sb *strings.Builder, enum protoreflect.EnumDescriptor, indent int) {
	indentStr := strings.Repeat("  ", indent)
	sb.WriteString(fmt.Sprintf("%senum %s {\n", indentStr, enum.Name()))

	for i := 0; i < enum.Values().Len(); i++ {
		value := enum.Values().Get(i)
		sb.WriteString(fmt.Sprintf("%s  %s = %d;\n", indentStr, value.Name(), value.Number()))
	}

	sb.WriteString(fmt.Sprintf("%s}\n", indentStr))
}

func generateMessage(sb *strings.Builder, msg protoreflect.MessageDescriptor, indent int) {
	// Skip internal map entry messages
	if msg.Options().(*descriptorpb.MessageOptions).GetMapEntry() {
		return
	}

	indentStr := strings.Repeat("  ", indent)
	sb.WriteString(fmt.Sprintf("%smessage %s {\n", indentStr, msg.Name()))

	// Generate nested enums
	for i := 0; i < msg.Enums().Len(); i++ {
		enum := msg.Enums().Get(i)
		generateEnum(sb, enum, indent+1)
		sb.WriteString("\n")
	}

	// Generate nested messages
	for i := 0; i < msg.Messages().Len(); i++ {
		nestedMsg := msg.Messages().Get(i)
		generateMessage(sb, nestedMsg, indent+1)
		sb.WriteString("\n")
	}

	// Collect and sort fields
	fields := make([]protoreflect.FieldDescriptor, msg.Fields().Len())
	for i := 0; i < msg.Fields().Len(); i++ {
		fields[i] = msg.Fields().Get(i)
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Number() < fields[j].Number()
	})

	// Generate sorted fields
	for _, field := range fields {
		fieldStr := generateField(field)
		sb.WriteString(fmt.Sprintf("%s  %s;\n", indentStr, fieldStr))
	}

	sb.WriteString(fmt.Sprintf("%s}\n", indentStr))
}

func generateField(field protoreflect.FieldDescriptor) string {
	var fieldStr string

	// Handle label (optional, repeated)
	if field.IsList() {
		fieldStr += "repeated "
	} else if field.HasOptionalKeyword() {
		fieldStr += "optional "
	}

	// Handle map fields
	if field.IsMap() {
		keyType := field.MapKey().Kind().String()
		valueType := getAppropriateTypeName(field.MapValue(), field.ParentFile())
		fieldStr += fmt.Sprintf("map<%s, %s>", keyType, valueType)
	} else {
		// Handle regular fields
		fieldStr += getAppropriateTypeName(field, field.ParentFile())
	}

	fieldStr += fmt.Sprintf(" %s = %d", field.Name(), field.Number())

	return fieldStr
}

func getAppropriateTypeName(field protoreflect.FieldDescriptor, currentFile protoreflect.FileDescriptor) string {
	if field.Kind() == protoreflect.MessageKind {
		return getMessageTypeName(field.Message(), currentFile)
	} else if field.Kind() == protoreflect.EnumKind {
		return getEnumTypeName(field.Enum(), currentFile)
	}
	return field.Kind().String()
}

func getMessageTypeName(message protoreflect.MessageDescriptor, currentFile protoreflect.FileDescriptor) string {
	if message.ParentFile() == currentFile {
		return string(message.Name())
	}
	return string(message.FullName())
}

func getEnumTypeName(enum protoreflect.EnumDescriptor, currentFile protoreflect.FileDescriptor) string {
	if enum.ParentFile() == currentFile {
		return string(enum.Name())
	}
	return string(enum.FullName())
}

func RunExample() {
	// Dynamically create a protobuf file definition with nested ENUM
	fileDesc := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("youtube/api/pfiinnertube/message.proto"),
		Package: proto.String("youtube.api.pfiinnertube"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("YoutubeApiInnertube"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("InnerTubeContext"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:     proto.String("client"),
								Number:   proto.Int32(1),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
								TypeName: proto.String(".youtube.api.pfiinnertube.YoutubeApiInnertube.ClientInfo"),
							},
							{
								Name:     proto.String("user"),
								Number:   proto.Int32(3),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
								TypeName: proto.String(".youtube.api.pfiinnertube.YoutubeApiInnertube.UserInfo"),
							},
							// Add other fields...
						},
					},
					{
						Name: proto.String("PlayerRequest"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:     proto.String("context"),
								Number:   proto.Int32(1),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
								TypeName: proto.String(".youtube.api.pfiinnertube.YoutubeApiInnertube.InnerTubeContext"),
							},
							// Add other fields...
						},
					},
					{
						Name:  proto.String("ClientInfo"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("UserInfo"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("SearchboxStats"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("LiteClientRequestData"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("UnpluggedBrowseOptions"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("ConsistencyToken"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("DeeplinkData"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("BrowseNotificationsParams"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("RecentUserEventInfo"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("DetectedActivityInfo"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("DeviceContextEvent"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("BrowseRequestSupportedMetadata"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("MdxContext"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("CustomTabContext"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("ProducerAssetRequestData"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("LatestContainerItemEventsInfo"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					{
						Name:  proto.String("ScrubContinuationClientData"),
						Field: []*descriptorpb.FieldDescriptorProto{
							// Add fields...
						},
					},
					// Add all other nested types...
				},
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{
						Name: proto.String("InlineSettingStatus"),
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{
								Name:   proto.String("UNKNOWN_INLINE_SETTING_STATUS"),
								Number: proto.Int32(0),
							},
						},
					},
					{
						Name: proto.String("BrowseRequestContext"),
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{
								Name:   proto.String("UNKNOWN_BROWSE_REQUEST_CONTEXT"),
								Number: proto.Int32(0),
							},
						},
					},
					// Add all other enum types...
				},
			},
		},
		Syntax: proto.String("proto3"),
	}

	// Convert the file descriptor to a FileDescriptor
	fd, err := protodesc.NewFile(fileDesc, nil)
	if err != nil {
		fmt.Printf("Error creating file descriptor: %v\n", err)
		return
	}

	// Generate the .proto file content
	protoContent := GenerateProtoFile(fd)

	// Write the content to a file
	err = os.WriteFile("example.proto", []byte(protoContent), 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}

	fmt.Println("Proto file generated successfully: example.proto")

}
