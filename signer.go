package iopipe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambdacontext"
)

type SignerRequest struct {
	ARN       string `json:"arn"`
	RequestID string `json:"requestId"`
	Timestamp int    `json:"timestamp"`
	Extension string `json:"extension"`
}

type SignerResponse struct {
	JWTAccess     string `json:"jwtAccess"`
	SignedRequest string `json:"signedRequest"`
	URL           string `json:"url"`
}

// GetSignerURL returns the URL for the signer in the specified region
func GetSignerURL(region string) string {
	supportedRegions := map[string]struct{}{
		"ap-northeast-1": struct{}{},
		"ap-southeast-2": struct{}{},
		"eu-west-1":      struct{}{},
		"us-east-1":      struct{}{},
		"us-east-2":      struct{}{},
		"us-west-1":      struct{}{},
		"us-west-2":      struct{}{},
	}

	if region == "mock" {
		return os.Getenv("MOCK_SERVER")
	}

	if _, exists := supportedRegions[region]; exists {
		return fmt.Sprintf("https://signer.%s.iopipe.com/", region)
	}

	return "https://signer.us-east-1.iopipe.com/"
}

// GetSignedRequest returns a signed request for uploading files to IOpipe
func GetSignedRequest(agent *Agent, context *lambdacontext.LambdaContext, extension string) (*SignerResponse, error) {
	var (
		err            error
		networkTimeout = 1 * time.Second
	)

	tr := &http.Transport{
		DisableKeepAlives: false,
		MaxIdleConns:      1, // is this equivalent to the maxCachedSessions in the js implementation
	}

	httpsClient := http.Client{Transport: tr, Timeout: networkTimeout}

	signerRequest := &SignerRequest{
		ARN:       context.InvokedFunctionArn,
		RequestID: context.AwsRequestID,
		Timestamp: int(time.Now().UnixNano() / 1e6),
		Extension: extension,
	}
	requestJSONBytes, _ := json.Marshal(signerRequest)
	agent.log.Debug("Signer request: ", string(requestJSONBytes))

	requestURL := GetSignerURL(os.Getenv("AWS_REGION"))

	req, err := http.NewRequest("POST", requestURL, bytes.NewReader(requestJSONBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", *agent.Config.Token)
	req.Header.Set("Content-Type", "application/json")

	res, err := httpsClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	bodyBytes, err := ioutil.ReadAll(res.Body)
	agent.log.Debug("Signer response: ", string(bodyBytes))
	if err != nil {
		return nil, err
	}

	var signerResponse *SignerResponse
	err = json.Unmarshal(bodyBytes, &signerResponse)
	if err != nil {
		return nil, err
	}

	return signerResponse, nil
}