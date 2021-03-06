package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/clbanning/mxj"
	"github.com/mitchellh/mapstructure"

	"github.com/TykTechnologies/tyk/apidef"
)

type ResponsetransformOptions struct {
	//FlushInterval time.Duration
}

type ResponseTransformMiddleware struct {
	Spec   *APISpec
	config ResponsetransformOptions
}

func (h *ResponseTransformMiddleware) Init(c interface{}, spec *APISpec) error {
	handler := ResponseTransformMiddleware{}

	if err := mapstructure.Decode(c, &h.config); err != nil {
		log.Error(err)
		return err
	}
	handler.Spec = spec
	return nil
}

func (h *ResponseTransformMiddleware) HandleResponse(rw http.ResponseWriter, res *http.Response, req *http.Request, ses *SessionState) error {
	_, versionPaths, _, _ := h.Spec.GetVersionData(req)
	found, meta := h.Spec.CheckSpecMatchesStatus(req.URL.Path, req.Method, versionPaths, TransformedResponse)
	if !found {
		return nil
	}
	tmeta := meta.(*TransformSpec)

	// Read the body:
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// Put into an interface:
	var bodyData map[string]interface{}
	switch tmeta.TemplateData.Input {
	case apidef.RequestXML:
		mxj.XmlCharsetReader = WrappedCharsetReader
		bodyData, err = mxj.NewMapXml(body) // unmarshal
		if err != nil {
			log.WithFields(logrus.Fields{
				"prefix":      "outbound-transform",
				"server_name": h.Spec.Proxy.TargetURL,
				"api_id":      h.Spec.APIID,
				"path":        req.URL.Path,
			}).Error("Error unmarshalling XML: ", err)
		}
	default: // apidef.RequestJSON
		json.Unmarshal(body, &bodyData)
	}

	// Apply to template
	var bodyBuffer bytes.Buffer
	if err = tmeta.Template.Execute(&bodyBuffer, bodyData); err != nil {
		log.WithFields(logrus.Fields{
			"prefix":      "outbound-transform",
			"server_name": h.Spec.Proxy.TargetURL,
			"api_id":      h.Spec.APIID,
			"path":        req.URL.Path,
		}).Error("Failed to apply template to request: ", err)
	}

	res.ContentLength = int64(bodyBuffer.Len())
	res.Header.Set("Content-Length", strconv.Itoa(bodyBuffer.Len()))
	res.Body = ioutil.NopCloser(&bodyBuffer)

	return nil
}
