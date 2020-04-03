/*
Copyright 2020 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aliyun_sls

import (
	"errors"
	"os"

	sls "github.com/aliyun/aliyun-log-go-sdk"
)

// Constants down below defines configurations to initialize a sls backend
// storage service, user should set sls environment variables in Dockerfile or
// deployment manifests files, for security user should better init environment
// variables by referencing Secret key as down below:
// spec:
//  containers:
//  - name: xxx-container
//    image: xxx
//    env:
//      - name: SLS_ENDPOINT
//        valueFrom:
//          secretKeyRef:
//            name: my-sls-secret
//            key: endpoint
const (
	EnvSLSEndpoint  = "SLS_ENDPOINT"
	EnvSLSKeyID     = "SLS_KEY_ID"
	EnvSLSKeySecret = "SLS_KEY_SECRET"
	EnvSLSProject   = "SLS_PROJECT"
	EnvSLSLogStore  = "SLS_LOG_STORE"
)

func GetSLSClient() (slsClient *sls.Client, project, logStore string, err error) {
	var (
		endpoint, keyID, keySecret string
	)

	if endpoint = os.Getenv(EnvSLSEndpoint); endpoint == "" {
		return nil, "", "", errors.New("empty sls endpoint")
	}
	if keyID = os.Getenv(EnvSLSKeyID); keyID == "" {
		return nil, "", "", errors.New("empty sls key id")
	}
	if keySecret = os.Getenv(EnvSLSKeySecret); keySecret == "" {
		return nil, "", "", errors.New("empty sls key secret")
	}
	if project = os.Getenv(EnvSLSProject); project == "" {
		return nil, "", "", errors.New("empty sls project name")
	}
	if logStore = os.Getenv(EnvSLSLogStore); logStore == "" {
		return nil, "", "", errors.New("empty sls log store")
	}

	return &sls.Client{
		Endpoint:        endpoint,
		AccessKeyID:     keyID,
		AccessKeySecret: keySecret,
	}, project, logStore, nil
}
