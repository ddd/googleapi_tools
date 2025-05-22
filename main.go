package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"req2proto/parser"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	logger               zerolog.Logger
	headerRe             = regexp.MustCompile(`:\s*`)
	messageRe            = regexp.MustCompile(`^((?:[a-z0-9_]+\.)*[a-z0-9_]+)\.([A-Z][A-Za-z.0-9_]+)$`)
	fieldDescRe          = regexp.MustCompile(`Invalid value at '(.+)' \((.*)\), (?:Base64 decoding failed for )?"?x?([^"]*)"?`)
	requiredFieldRe      = regexp.MustCompile(`Missing required field (.+) at '([^']+)'`)
	packageFDProtoMap    = make(map[string]*descriptorpb.FileDescriptorProto)
	packageDependencyMap = make(map[string][]string)
)

var typeMap = map[string]*descriptorpb.FieldDescriptorProto_Type{
	"TYPE_STRING":   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
	"TYPE_BOOL":     descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(),
	"TYPE_INT64":    descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
	"TYPE_UINT64":   descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
	"TYPE_INT32":    descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
	"TYPE_UINT32":   descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum(),
	"TYPE_DOUBLE":   descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(),
	"TYPE_FLOAT":    descriptorpb.FieldDescriptorProto_TYPE_FLOAT.Enum(),
	"TYPE_BYTES":    descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(),
	"TYPE_FIXED64":  descriptorpb.FieldDescriptorProto_TYPE_FIXED64.Enum(),
	"TYPE_FIXED32":  descriptorpb.FieldDescriptorProto_TYPE_FIXED32.Enum(),
	"TYPE_SINT64":   descriptorpb.FieldDescriptorProto_TYPE_SINT64.Enum(),
	"TYPE_SINT32":   descriptorpb.FieldDescriptorProto_TYPE_SINT32.Enum(),
	"TYPE_SFIXED64": descriptorpb.FieldDescriptorProto_TYPE_SFIXED64.Enum(),
	"TYPE_SFIXED32": descriptorpb.FieldDescriptorProto_TYPE_SFIXED32.Enum(),
}

type headerSliceFlag []string

func (h *headerSliceFlag) String() string {
	return fmt.Sprint(*h)
}

func (h *headerSliceFlag) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func cleanupDuplicateFields(fdproto *descriptorpb.FileDescriptorProto, verbose bool) {
	// Create maps for top-level enum names
	enumNames := make(map[string]bool)

	// Populate top-level enum names first
	for _, enum := range fdproto.EnumType {
		enumNames[*enum.Name] = true
	}

	// Check and remove conflicting top-level messages
	var newMessageType []*descriptorpb.DescriptorProto
	for _, msgType := range fdproto.MessageType {
		if enumNames[*msgType.Name] {
			if verbose {
				logger.Debug().Str("message", *msgType.Name).Msg("Removed top-level message as it conflicts with an enum")
			}
		} else {
			cleanupMessageType(msgType, verbose)
			newMessageType = append(newMessageType, msgType)
		}
	}
	fdproto.MessageType = newMessageType

}

func cleanupMessageType(msgType *descriptorpb.DescriptorProto, verbose bool) {
	// Create a map of enum names
	enumNames := make(map[string]bool)

	// Populate enum names first
	for _, enum := range msgType.EnumType {
		enumNames[*enum.Name] = true
	}

	// Check and remove conflicting nested types
	var newNestedType []*descriptorpb.DescriptorProto
	for _, nestedType := range msgType.NestedType {
		if enumNames[*nestedType.Name] {
			if verbose {
				logger.Debug().Str("message", *nestedType.Name).Msg("Removed nested message as it conflicts with an enum")
			}
		} else {
			newNestedType = append(newNestedType, nestedType)
		}
	}
	msgType.NestedType = newNestedType

	// Recursively clean up nested message types
	for _, nestedType := range msgType.NestedType {
		cleanupMessageType(nestedType, verbose)
	}
}

func getOrCreateMessageDescriptor(fileDesc *descriptorpb.FileDescriptorProto, messageName string) (*descriptorpb.DescriptorProto, *descriptorpb.EnumDescriptorProto, error) {
	parts := strings.Split(messageName, ".")
	var currentMessage *descriptorpb.DescriptorProto
	var currentMessages *[]*descriptorpb.DescriptorProto = &fileDesc.MessageType
	var currentEnums *[]*descriptorpb.EnumDescriptorProto = &fileDesc.EnumType

	for i, part := range parts {
		if i == len(parts)-1 {
			// Check if the last part is an enum
			if enum := findEnum(currentEnums, part); enum != nil {
				return nil, enum, nil
			}
		}

		currentMessage = findOrCreateMessage(currentMessages, part)

		if i < len(parts)-1 {
			// If we're not at the last part, we need to go deeper
			currentMessages = &currentMessage.NestedType
			currentEnums = &currentMessage.EnumType
		}
	}

	return currentMessage, nil, nil
}

func findOrCreateMessage(messages *[]*descriptorpb.DescriptorProto, name string) *descriptorpb.DescriptorProto {
	for _, msg := range *messages {
		if msg.GetName() == name {
			return msg
		}
	}

	// If the message doesn't exist, create it
	newMessage := &descriptorpb.DescriptorProto{
		Name: proto.String(name),
	}
	*messages = append(*messages, newMessage)
	return newMessage
}

func findEnum(enums *[]*descriptorpb.EnumDescriptorProto, name string) *descriptorpb.EnumDescriptorProto {
	for _, enum := range *enums {
		if enum.GetName() == name {
			return enum
		}
	}
	return nil
}

type MsgChData struct {
	Package               string
	Message               string
	Index                 []int
	DescProto             *descriptorpb.DescriptorProto
	ParentDescProto       *descriptorpb.DescriptorProto
	RequiredFieldsToLabel []string
}

func monitorAndCloseChannel(msgCh chan MsgChData) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			if len(msgCh) > 0 {
				timer.Reset(10 * time.Second)
			}
		case <-timer.C:
			close(msgCh)
			return
		}
	}
}

// This function recieves fdProto and index of messages to probe further fields in
func probeNestedMessageWorker(msgCh chan MsgChData, method string, url string, headers map[string]string, maxDepth int, verbose bool) {

	for msgChData := range msgCh {

		// probe for int violations
		payload := genPayload(msgChData.Index, "int")
		intViolations, err := probeAPI(method, url, headers, payload)
		if err != nil {
			logger.Fatal().Err(err).Msg("error when probing api")
		}

		// probe for str violations
		payload = genPayload(msgChData.Index, "str")
		violations, err := probeAPI(method, url, headers, payload)
		if err != nil {
			logger.Fatal().Err(err).Msg("error when probing api")
		}

		// add all violations together
		violations = append(violations, intViolations...)

		// TODO: add mutex locks everywhere when iterating and appending

		alreadyPresentFields := make(map[int]struct{})
		for _, field := range msgChData.DescProto.Field {
			alreadyPresentFields[int(*field.Number)] = struct{}{}
		}

		// first, we have to loop through violations and find if there's any required field errors. requiredFieldMap is a map of full field name of message and array of required fields
		requiredFieldMap := make(map[string][]string, 300)
		for _, i := range violations {
			if strings.HasPrefix(i.Description, "Missing required field") {

				x := requiredFieldRe.FindStringSubmatch(i.Description)
				requiredFields, ok := requiredFieldMap[i.Field]
				if !ok {
					requiredFieldMap[i.Field] = []string{x[1]}
				} else {
					requiredFieldMap[i.Field] = append(requiredFields, x[1])
				}
			}
		}

		addedFields := make(map[string]struct{}, 100)
		for _, i := range violations {

			// enum
			if i.Description == "Invalid value (), Unexpected list for single non-message field." || i.Description == "Invalid value (), List is not message or group type." {
				// if enum, we find parent, then set it's field Type and TypeName. after that, we append an entry to EnumType.
				x := strings.Split(msgChData.Message, ".")

				lastIndex := msgChData.Index[len(msgChData.Index)-1]

				for _, i := range msgChData.ParentDescProto.Field {
					if int(*i.Number) == lastIndex {
						if verbose {
							logger.Debug().Str("field_name", *i.Name).Str("package", msgChData.Package).Str("message", msgChData.Message).Msg("updated type to enum")
						}
						i.Type = descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum()
						newTypeName := "." + msgChData.Package + "." + msgChData.Message
						i.TypeName = &newTypeName
					}
				}

				// If the parent is the FileDescriptorProto
				if len(x) == 1 {
					actualParentDesc := packageFDProtoMap[msgChData.Package]

					exists := false
					for _, i := range actualParentDesc.EnumType {
						if *i.Name == msgChData.Message {
							exists = true
							break
						}
					}

					if !exists {
						actualParentDesc.EnumType = append(actualParentDesc.EnumType, &descriptorpb.EnumDescriptorProto{
							Name: proto.String(msgChData.Message),
							Value: []*descriptorpb.EnumValueDescriptorProto{
								{Name: proto.String(convertToUnknownType(msgChData.Message)), Number: proto.Int32(0)},
							},
						})
					}

				} else {
					actualParentDesc, _, err := getOrCreateMessageDescriptor(packageFDProtoMap[msgChData.Package], strings.Join(x[:len(x)-1], "."))
					if err != nil {
						panic(err)
					}

					exists := false
					for _, i := range actualParentDesc.EnumType {
						if *i.Name == x[len(x)-1] {
							exists = true
							break
						}
					}

					if !exists {
						actualParentDesc.EnumType = append(actualParentDesc.EnumType, &descriptorpb.EnumDescriptorProto{
							Name: proto.String(x[len(x)-1]),
							Value: []*descriptorpb.EnumValueDescriptorProto{
								{Name: proto.String(convertToUnknownType(x[len(x)-1])), Number: proto.Int32(0)},
							},
						})
					}
				}
				break
			}

			// required field, we settled this before, so we can skip
			if strings.HasPrefix(i.Description, "Missing required field") {
				continue
			}

			z := strings.Split(i.Field, ".")
			fieldName := z[len(z)-1]
			matches := fieldDescRe.FindStringSubmatch(i.Description)
			if len(matches) < 3 {
				logger.Error().Str("description", i.Description).Str("message", msgChData.Message).Msg("unable to parse violation error description")
				continue
			}

			number, _ := strconv.Atoi(matches[3])

			// repeated
			if strings.HasSuffix(fieldName, "]") {
				// if repeated, we find parent, and then set this as repeated flag
				x := strings.Split(msgChData.Message, ".")

				// find the message's field, and set the label to repeated
				for _, i := range msgChData.ParentDescProto.Field {
					if *i.Name == x[len(x)-1] {
						i.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
					}
				}

				if verbose {
					logger.Debug().Str("field_name", fieldName).Str("parent_message", *msgChData.ParentDescProto.Name).Str("package", msgChData.Package).Str("index", fmt.Sprint(msgChData.Index)).Msg("set field as repeated")
				}

				// after that, we append 0 to index and send back to msgCh, then break out of the violations loop
				if maxDepth < 0 || !(len(msgChData.Index) == maxDepth) {
					msgCh <- MsgChData{Package: msgChData.Package, Message: msgChData.Message, DescProto: msgChData.DescProto, ParentDescProto: msgChData.DescProto, Index: append(msgChData.Index, 1)}
				}
				break
			}

			// field is not a message
			if strings.HasPrefix(matches[2], "TYPE_") {
				_, ok := alreadyPresentFields[number]
				if !ok {
					label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()
					for _, requiredField := range msgChData.RequiredFieldsToLabel {
						if requiredField == fieldName {
							label = descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum()
							packageFDProtoMap[msgChData.Package].Syntax = proto.String("proto2")
						}
					}

					addedFields[fieldName] = struct{}{}
					msgChData.DescProto.Field = append(msgChData.DescProto.Field, &descriptorpb.FieldDescriptorProto{
						Name:     proto.String(fieldName),
						Number:   proto.Int32(int32(number)),
						Label:    label,
						Type:     typeMap[matches[2]],
						JsonName: proto.String(fieldName),
					})
					alreadyPresentFields[number] = struct{}{}
				}
			} else {
				_, ok := alreadyPresentFields[number]
				if !ok {
					z := strings.Split(matches[2], ".")
					nestedMessageName := z[len(z)-1]

					// getting the package and message name from the googleapi type
					//fmt.Println(matches[2])
					x := messageRe.FindStringSubmatch(strings.Split(matches[2], "type.googleapis.com/")[1])
					var packageName, fullMessageName string
					if x != nil {
						packageName = x[1]
						fullMessageName = x[2]
					} else {
						// if we can't find a package name, we just put it under google
						packageName = "google"
						fullMessageName = strings.Split(matches[2], "type.googleapis.com/")[1]
						nestedMessageName = fullMessageName
					}

					var descProto *descriptorpb.DescriptorProto
					fdproto, ok := packageFDProtoMap[packageName]
					if !ok {
						// we can't find a fdproto for that package, so we make a new one
						packageFDProtoMap[packageName] = &descriptorpb.FileDescriptorProto{
							Name:    proto.String(strings.Replace(packageName, ".", "/", -1) + "/message.proto"),
							Syntax:  proto.String("proto3"),
							Package: proto.String(packageName),
							MessageType: []*descriptorpb.DescriptorProto{
								{
									Name: proto.String(nestedMessageName),
								},
							},
						}

						fdproto = packageFDProtoMap[packageName]
						descProto = fdproto.MessageType[0]

					} else {
						// the fdproto for that package exists, so we find if the message exists in there, if it doesn't, create it
						var enum *descriptorpb.EnumDescriptorProto
						descProto, enum, err = getOrCreateMessageDescriptor(fdproto, fullMessageName)
						if err != nil {
							panic(err)
						}

						// skip, as there's already an enum with this name
						if enum != nil {
							continue
						}
					}

					// adding dependency of package if it's on another file
					if packageName != msgChData.Package {
						dependencyFileName := strings.Replace(packageName, ".", "/", -1) + "/message.proto"
						alreadyAdded := false
						for _, i := range packageDependencyMap[msgChData.Package] {
							if i == dependencyFileName {
								alreadyAdded = true
								break
							}
						}
						if !alreadyAdded {
							packageDependencyMap[msgChData.Package] = append(packageDependencyMap[msgChData.Package], dependencyFileName)
						}

					}

					// if we found google.protobuf.Any, we don't need to probe it
					if matches[2] == "type.googleapis.com/google.protobuf.Any" {
						exists := false
						for _, i := range packageFDProtoMap["google.protobuf"].MessageType {
							if *i.Name == "Any" && i.Field != nil && *i.Field[0].Name == "type_url" {
								exists = true
								break
							}
						}

						if !exists {

							for _, i := range packageFDProtoMap["google.protobuf"].MessageType {
								if *i.Name == "Any" {
									i.Field = append(i.Field, []*descriptorpb.FieldDescriptorProto{
										{
											Name:     proto.String("type_url"),
											Number:   proto.Int32(1),
											Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
											Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum().Enum(),
											JsonName: proto.String("type_url"),
										},
										{
											// this is a guess as to what the internal name is, no clue honestly.
											Name:     proto.String("data"),
											Number:   proto.Int32(2),
											Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
											Type:     descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(),
											JsonName: proto.String("data"),
										},
									}...)
								}
							}
						}
					} else {
						// send the descProto to msgCh so that it will be probed next
						if maxDepth < 0 || !(len(msgChData.Index) == maxDepth) {
							newIndex := append(msgChData.Index, number)
							requiredFields := requiredFieldMap[i.Field]
							msgCh <- MsgChData{Package: packageName, Message: fullMessageName, DescProto: descProto, ParentDescProto: msgChData.DescProto, Index: newIndex, RequiredFieldsToLabel: requiredFields}
						}
					}

					label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()
					for _, requiredField := range msgChData.RequiredFieldsToLabel {
						if requiredField == fieldName {
							label = descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum()
							packageFDProtoMap[packageName].Syntax = proto.String("proto2")
						}
					}

					addedFields[fieldName] = struct{}{}
					msgChData.DescProto.Field = append(msgChData.DescProto.Field, &descriptorpb.FieldDescriptorProto{
						Name:     proto.String(fieldName),
						Number:   proto.Int32(int32(number)),
						Label:    label,
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String("." + strings.Split(matches[2], "type.googleapis.com/")[1]),
						JsonName: proto.String(fieldName),
					})
					alreadyPresentFields[number] = struct{}{}
				}

			}

		}

	}

}

func modifyAltParameter(inputURL string) string {
	// Parse the URL
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		panic(err)
	}

	// Get the query parameters
	values := parsedURL.Query()

	// Check if alt parameter exists
	altValue := values.Get("alt")
	if altValue == "" {
		// If alt doesn't exist, add alt=json
		values.Set("alt", "json")
	} else if altValue != "json" {
		// If alt exists but isn't json, replace it with json
		values.Set("alt", "json")
	}

	// Set the new query string
	parsedURL.RawQuery = values.Encode()
	return parsedURL.String()
}

func main() {
	// Define flags
	method := flag.String("X", "POST", "HTTP method (GET or POST)")
	url := flag.String("u", "", "URL to send the request to")
	maxDepth := flag.Int("d", -1, "Maximum depth to probe (unlimited: -1)")
	outputDir := flag.String("o", "output", "Directory for .proto files to be output (can be full or relative path)")
	verbose := flag.Bool("v", false, "Verbose mode")
	reqMessageName := flag.String("p", "google.example.Request", "Full type name for request, usually similar to gRPC name (ex. google.internal.people.v2.minimal.ListRankedTargetsRequest)")

	// Use a custom flag for headers
	var headers headerSliceFlag
	flag.Var(&headers, "H", "Headers in format 'Key: Value' (can be used multiple times)")

	flag.Parse()

	logFile, _ := os.OpenFile("latest.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	defer logFile.Close()

	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
	multi := zerolog.MultiLevelWriter(consoleWriter, logFile)
	logger = zerolog.New(multi).With().Timestamp().Logger()

	if *url == "" {
		panic("no url supplied!")
	}

	*url = modifyAltParameter(*url)

	headersMap := make(map[string]string, 20)
	for _, i := range headers {
		j := headerRe.Split(i, 2)
		headersMap[j[0]] = j[1]
	}

	payload := genPayload(nil, "str")
	s1, _, err := testAPI(*method, *url, headersMap, payload)
	if err != nil {
		logger.Fatal().Err(err)
	}
	payload = genPayload(nil, "int")
	s2, r2, err := testAPI(*method, *url, headersMap, payload)
	if err != nil {
		logger.Fatal().Err(err)
	}

	if s1 != 400 && s2 != 400 {
		logger.Fatal().Int("status", s2).Str("resp", string(r2)).Msg("unknown status code")
	}

	msgCh := make(chan MsgChData, 1000)

	x := messageRe.FindStringSubmatch(*reqMessageName)
	packageName := x[1]
	messageName := x[2]

	packageFDProtoMap[packageName] = &descriptorpb.FileDescriptorProto{
		Name:    proto.String(strings.Replace(packageName, ".", "/", -1) + "/message.proto"),
		Syntax:  proto.String("proto3"),
		Package: proto.String(packageName),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name:       proto.String(messageName),
				Field:      []*descriptorpb.FieldDescriptorProto{},
				NestedType: []*descriptorpb.DescriptorProto{},
			},
		},
	}

	fdproto := packageFDProtoMap[packageName]
	descProto := fdproto.MessageType[0]

	// ParentDescProto is nil for initial
	msgCh <- MsgChData{Package: packageName, Message: messageName, Index: []int{}, DescProto: descProto}

	go monitorAndCloseChannel(msgCh)
	probeNestedMessageWorker(msgCh, *method, *url, headersMap, *maxDepth, *verbose)

	for p, fdproto := range packageFDProtoMap {
		fdproto.Dependency = append(fdproto.Dependency, packageDependencyMap[p]...)
	}

	fileDescSet := &descriptorpb.FileDescriptorSet{}
	processFileDescriptors(packageFDProtoMap)
	for _, i := range packageFDProtoMap {
		cleanupDuplicateFields(i, *verbose)
		if *verbose {
			text := prototext.Format(i)
			logger.Debug().Msg(text)
		}
		fileDescSet.File = append(fileDescSet.File, i)
	}

	fileOptions := protodesc.FileOptions{AllowUnresolvable: true}
	files := &protoregistry.Files{}

	for _, fdProto := range fileDescSet.File {
		descriptor, err := fileOptions.New(fdProto, files)
		if err != nil {
			log.Printf("Error creating FileDescriptor for %s: %v\n", *fdProto.Name, err)
			continue
		}

		if err := files.RegisterFile(descriptor); err != nil {
			log.Printf("Error registering file %s: %v\n", *fdProto.Name, err)
			continue
		}

		fileContent := parser.GenerateProtoFile(descriptor)

		fileName := *outputDir + "/" + *fdProto.Name
		writeFile([]byte(fileContent), fileName)

		if *verbose {
			logger.Debug().Str("file", fileName).Msg("proto file generated successfully")
		}
	}

}
