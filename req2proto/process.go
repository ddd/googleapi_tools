package main

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func processFileDescriptors(fdMap map[string]*descriptorpb.FileDescriptorProto) {
	enumMap := make(map[string]bool)

	// First pass: Collect all enum types across all packages
	for _, fd := range fdMap {
		collectEnumTypes(fd, enumMap)
	}

	// Second pass: Update field types and handle duplicate field names
	for _, fd := range fdMap {
		updateFieldTypes(fd, enumMap)
	}
}

func collectEnumTypes(fd *descriptorpb.FileDescriptorProto, enumMap map[string]bool) {
	// Process root-level enums
	for _, enum := range fd.EnumType {
		fullName := fmt.Sprintf("%s.%s", *fd.Package, *enum.Name)
		enumMap[fullName] = true
	}

	// Process enums in messages
	for _, msg := range fd.MessageType {
		collectMessageEnumTypes(msg, *fd.Package, enumMap)
	}
}

func collectMessageEnumTypes(msg *descriptorpb.DescriptorProto, parentPath string, enumMap map[string]bool) {
	currentPath := fmt.Sprintf("%s.%s", parentPath, *msg.Name)

	// Process enums in this message
	for _, enum := range msg.EnumType {
		fullName := fmt.Sprintf("%s.%s", currentPath, *enum.Name)
		enumMap[fullName] = true
	}

	// Process nested messages
	for _, nestedMsg := range msg.NestedType {
		collectMessageEnumTypes(nestedMsg, currentPath, enumMap)
	}
}

func updateFieldTypes(fd *descriptorpb.FileDescriptorProto, enumMap map[string]bool) {
	updateMessageFieldTypes(fd.MessageType, *fd.Package, enumMap)
}

func updateMessageFieldTypes(messages []*descriptorpb.DescriptorProto, parentPath string, enumMap map[string]bool) {
	for _, msg := range messages {
		currentPath := fmt.Sprintf("%s.%s", parentPath, *msg.Name)
		fieldNames := make(map[string]bool)

		for _, field := range msg.Field {
			// Check for duplicate field names
			if fieldNames[*field.Name] {
				newName := fmt.Sprintf("%s_%s", *field.Name, generateRandomString())
				field.Name = proto.String(newName)
			}
			fieldNames[*field.Name] = true

			if field.TypeName != nil {
				fullTypeName := strings.TrimPrefix(*field.TypeName, ".")
				if enumMap[fullTypeName] {
					// Change field type to enum
					field.Type = descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum()
				}
			}
		}

		// Recursively update nested messages
		updateMessageFieldTypes(msg.NestedType, currentPath, enumMap)
	}
}
