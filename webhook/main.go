package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog"
)

const (
	ListenAddr    string = ":8443"
	RequiredLabel string = "appid"
)

var (
	codecs = serializer.NewCodecFactory(runtime.NewScheme())
)

func main() {
	// Set up HTTP request handlers
	http.HandleFunc("/validate", validatePod)
	http.HandleFunc("/mutate", mutatePod)

	// Start the server and listen for HTTPS connections
	err := http.ListenAndServeTLS(ListenAddr, "/opt/tls.crt", "/opt/tls.key", nil)
	if err != nil {
		klog.Fatalln("Failed to start the server:", err)
	}
	klog.Infoln("Server started successfully ", ListenAddr)
}

// admissionReviewFromRequest extracts the AdmissionReview object from the HTTP request.
func admissionReviewFromRequest(r *http.Request, deserializer runtime.Decoder) (*admissionv1.AdmissionReview, error) {
	if r.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("expected application/json content-type")
	}

	var body []byte
	if r.Body != nil {
		requestData, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		body = requestData
	}

	admissionReviewRequest := &admissionv1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, admissionReviewRequest); err != nil {
		return nil, err
	}
	return admissionReviewRequest, nil
}

// validatePod is an HTTP request handler for validating Pod objects.
func validatePod(w http.ResponseWriter, r *http.Request) {
	deserializer := codecs.UniversalDeserializer()
	admissionReviewRequest, err := admissionReviewFromRequest(r, deserializer)
	if err != nil {
		msg := fmt.Sprintf("error getting admission review from request: %v", err)
		klog.Errorln(msg)
		w.WriteHeader(400)
		_, _ = w.Write([]byte(msg))
		return
	}

	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if admissionReviewRequest.Request.Resource != podResource {
		msg := fmt.Sprintf("did not receive pod, got %s", admissionReviewRequest.Request.Resource.Resource)
		klog.Errorln(msg)
		w.WriteHeader(400)
		_, _ = w.Write([]byte(msg))
		return
	}

	rawRequest := admissionReviewRequest.Request.Object.Raw
	pod := corev1.Pod{}
	if _, _, err := deserializer.Decode(rawRequest, nil, &pod); err != nil {
		msg := fmt.Sprintf("error decoding raw pod: %v", err)
		klog.Errorln(msg)
		w.WriteHeader(500)
		_, _ = w.Write([]byte(msg))
		return
	}

	klog.Infof("Received request: %s %s => %s/%s", r.Method, r.URL.Path, pod.Namespace, pod.Name)

	admissionResponse := &admissionv1.AdmissionResponse{}
	admissionResponse.Allowed = true

	// Check if the Pod has the required label
	if _, ok := pod.Labels[RequiredLabel]; !ok {
		admissionResponse.Allowed = false
		admissionResponse.Result = &metav1.Status{
			Message: fmt.Sprintf("Pod must have the %q label!\n", RequiredLabel),
		}
	}

	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.SetGroupVersionKind(admissionReviewRequest.GroupVersionKind())
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.Response.UID = admissionReviewRequest.Request.UID

	resp, err := json.Marshal(admissionReviewResponse)
	if err != nil {
		msg := fmt.Sprintf("error marshalling response json: %v", err)
		klog.Errorln(msg)
		w.WriteHeader(500)
		_, _ = w.Write([]byte(msg))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resp)
}

// mutatePod is an HTTP request handler for mutating Pod objects.
func mutatePod(w http.ResponseWriter, r *http.Request) {
	deserializer := codecs.UniversalDeserializer()
	admissionReviewRequest, err := admissionReviewFromRequest(r, deserializer)
	if err != nil {
		msg := fmt.Sprintf("error getting admission review from request: %v", err)
		klog.Errorln(msg)
		w.WriteHeader(400)
		_, _ = w.Write([]byte(msg))
		return
	}

	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if admissionReviewRequest.Request.Resource != podResource {
		msg := fmt.Sprintf("did not receive pod, got %s", admissionReviewRequest.Request.Resource.Resource)
		klog.Errorln(msg)
		w.WriteHeader(400)
		_, _ = w.Write([]byte(msg))
		return
	}

	rawRequest := admissionReviewRequest.Request.Object.Raw
	pod := corev1.Pod{}
	if _, _, err := deserializer.Decode(rawRequest, nil, &pod); err != nil {
		msg := fmt.Sprintf("error decoding raw pod: %v", err)
		klog.Errorln(msg)
		w.WriteHeader(500)
		_, _ = w.Write([]byte(msg))
		return
	}

	klog.Infof("Received request: %s %s => %s/%s", r.Method, r.URL.Path, pod.Namespace, pod.Name)

	admissionResponse := &admissionv1.AdmissionResponse{}
	patchType := admissionv1.PatchTypeJSONPatch
	admissionResponse.Allowed = true
	admissionResponse.PatchType = &patchType
	admissionResponse.Patch = []byte(`[{
		"op": "add",
		"path": "/spec/initContainers",
		"value": [
			{
				"name": "init-container",
				"image": "busybox",
				"command": ["sleep", "120"]
			}
		]
	}]`)

	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.SetGroupVersionKind(admissionReviewRequest.GroupVersionKind())
	admissionReviewResponse.Response = admissionResponse
	admissionReviewResponse.Response.UID = admissionReviewRequest.Request.UID

	resp, err := json.Marshal(admissionReviewResponse)
	if err != nil {
		msg := fmt.Sprintf("error marshalling response json: %v", err)
		klog.Errorln(msg)
		w.WriteHeader(500)
		_, _ = w.Write([]byte(msg))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resp)
}
