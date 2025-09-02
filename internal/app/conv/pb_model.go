package conv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var marshalOptions = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: true,
}

var unmarshalOptions = protojson.UnmarshalOptions{
	AllowPartial:   false,
	DiscardUnknown: true,
}

func ModelToPb(ctx context.Context, model interface{}, protoMsg proto.Message) error {
	if model == nil {
		return errors.New("model to pb error: protoMsg == nil")
	} else if protoMsg == nil {
		return errors.New("model to pb error: protoMsg == nil")
	}

	//logger.Debugf(ctx,
	//	"model '%v' to pb: model != nil :'%+v'", protoMsg.ProtoReflect().Type().Descriptor().Name(), model)

	data, err := json.Marshal(model)
	if err != nil {
		err = errors.Wrapf(err,
			"model '%v' to pb marshal error", protoMsg.ProtoReflect().Type().Descriptor().Name())
		logger.Errorf(ctx, "Errr: %v", err)

		return err
	}

	// logger.Debugf("Marshal '%v' data: '%+v'",
	//	protoMsg.ProtoReflect().Type().Descriptor().Name(), string(data))

	err = unmarshalOptions.Unmarshal(data, protoMsg)
	if err != nil {
		err = errors.Wrapf(err, "model to pb '%v' unmarshal error; data: '%.100v...';",
			protoMsg.ProtoReflect().Type().Descriptor().Name(), string(data))

		logger.Debugf(ctx, "model '%v' to pb:'%+v'",
			protoMsg.ProtoReflect().Type().Descriptor().Name(), protoMsg)
		if strings.Contains(err.Error(), ": unexpected token ") {
			// logger.Infof(
			//	"Crutch for type '%v' conversion", protoMsg.ProtoReflect().Type().Descriptor().Name())
			// logger.Debugf("Crutch    model:'%+v'", model)
			// logger.Debugf("Crutch protoMsg:'%+v'", protoMsg)

			// костыль из-за Error preparing the Run entity:
			//'Error preparing the Run entity, runModelToPB:'proto: syntax error (line 1:121): unexpected token

			return nil
		} // fixme: unexpected token, теряются все поля после поля которое приводит к ошибке, сложное поле должно быть в конце

		logger.Errorf(ctx, "Unmarshal error: %v", err)

		return err
	}

	// logger.Debugf("Unmarshal '%v' ProtoMsg: '%+v'",
	//	protoMsg.ProtoReflect().Type().Descriptor().Name(), protoMsg)

	return nil
}

func PbToModel(ctx context.Context, model interface{}, protoMsg proto.Message) (message string) {
	if protoMsg == nil {
		return "pb to model error: protoMsg == nil"
	}

	// logger.Debugf("'%v' pb to model '%v'",
	//	protoMsg.ProtoReflect().Type().Descriptor().Name(), protoMsg)

	data, err := marshalOptions.Marshal(protoMsg)
	if err != nil {
		message = fmt.Sprintf("pb '%v' to model marshal error: '%v'",
			protoMsg.ProtoReflect().Type().Descriptor().Name(), err)
		logger.Errorf(ctx, "Mess %v", message)

		return message
	}

	// logger.Debugf("PbToModel: '%v' -> '%v'",
	//	protoMsg.ProtoReflect().Type().Descriptor().Name(), string(data))

	if er := json.Unmarshal(data, model); er != nil {
		message = fmt.Sprintf("pb '%v' to model unmarshal error: '%+v',  data: '%.200v...'",
			protoMsg.ProtoReflect().Type().Descriptor().Name(), er, string(data))
		if strings.Contains(message, "cannot unmarshal array into Go struct field Run.script_runs of type string") {
			message = "" // fixme: костыль из-за cannot unmarshal array into Go struct field Run.script_runs of type string
			return message
		}

		if strings.Contains(
			message,
			"cannot unmarshal array into Go struct field SimpleScript.query_params of type string",
		) {
			message = "" // fixme: костыль из-за cannot unmarshal array into Go struct field SimpleScript.query_params of type string
			return message
		}

		if strings.Contains(message, "cannot unmarshal array into Go struct field GroupsPage.groups of type string") {
			message = "" // fixme: костыль из-за cannot unmarshal array into Go struct field GroupsPage.groups of type string
			return message
		}
		if strings.Contains(
			message,
			"cannot unmarshal object into Go struct field Script.additional_env of type string",
		) {
			message = "" // fixme: костыль из-за cannot unmarshal array into Go struct field Script.additional_env of type string
			return message
		}
		if strings.Contains(message, "cannot unmarshal array into Go struct field Scenario.selectors of type string") {
			message = "" // fixme: костыль из-за cannot unmarshal array into Go struct field Scenario.selectors of type string
			return message
		}

		logger.Errorf(ctx, "'%v' -> '%v'",
			protoMsg.ProtoReflect().Type().Descriptor().Name(), message)
	}

	return message
}

func PbToString(ctx context.Context, protoMsg proto.Message) (model string, err error) {
	if protoMsg == nil {
		return "", fmt.Errorf("pb to string error: protoMsgs == nil")
	}

	logger.Debugf(ctx, "'%v' pb to string '%v'",
		protoMsg.ProtoReflect().Type().Descriptor().Name(), protoMsg)

	data, err := marshalOptions.Marshal(protoMsg)
	if err != nil {
		//err = errors.Wrapf(err, "pb '%v' to string marshal error. ",
		//	protoMsgs.ProtoReflect().Type().Descriptor().Name())
		logger.Errorf(ctx, "PbToString error: %v", err)
	}

	logger.Debugf(ctx, "PbToString: '%v' -> '%v'",
		protoMsg.ProtoReflect().Type().Descriptor().Name(), string(data))

	return string(data), err
}
