// +build coprocess

package main

import (
	"encoding/json"

	"github.com/TykTechnologies/tyk/apidef"
	"github.com/TykTechnologies/tyk/config"
)

// Constant for event system.
const (
	EH_CoProcessHandler apidef.TykEventHandlerName = "cp_dynamic_handler"
)

type CoProcessEventHandler struct {
	conf     map[string]interface{}
	Spec     *APISpec
	SpecJSON json.RawMessage
}

type CoProcessEventWrapper struct {
	Event    config.EventMessage `json:"message"`
	Handler  string              `json:"handler_name"`
	SpecJSON *json.RawMessage    `json:"spec"`
}

func (l *CoProcessEventHandler) Init(handlerConf interface{}) error {
	l.conf = handlerConf.(map[string]interface{})

	// Set the VM globals
	globalVals := JSVMContextGlobal{
		APIID: l.Spec.APIID,
		OrgID: l.Spec.OrgID,
	}

	gValAsJSON, err := json.Marshal(globalVals)
	if err != nil {
		log.Error("Failed to marshal globals! ", err)
	}

	l.SpecJSON = json.RawMessage(gValAsJSON)
	return nil
}

func (l *CoProcessEventHandler) HandleEvent(em config.EventMessage) {
	// 1. Get the methodName for the Event Handler
	methodName := l.conf["name"].(string)

	eventWrapper := CoProcessEventWrapper{
		Event:    em,
		Handler:  methodName,
		SpecJSON: &l.SpecJSON,
	}

	// 2. JSON-encode the event data object
	msgAsJSON, err := json.Marshal(eventWrapper)
	if err != nil {
		log.Error("Failed to encode event data: ", err)
		return
	}

	if GlobalDispatcher != nil {
		GlobalDispatcher.DispatchEvent(msgAsJSON)
	}
}
