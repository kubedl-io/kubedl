package model

type OAuthInfo struct {

	// aliCloud third-part app id
	AppId string `json:"appId"`
	// aliCloud third-part app secret
	AppSecret string `json:"appSecret"`

	UserInfo UserInfo `json:"userInfo"`
}

// GetOauthInfo from configmap
func GetOauthInfo(oauthConfig map[string]string) OAuthInfo {
	if oauthConfig == nil {
		return OAuthInfo{}
	}

	OAuthInfo := OAuthInfo{
		AppId:     oauthConfig["appId"],
		AppSecret: oauthConfig["appSecret"],
		UserInfo: UserInfo{
			Aid:       oauthConfig["aid"],
			Uid:       oauthConfig["uid"],
			Name:      oauthConfig["name"],
			LoginName: oauthConfig["loginName"],
			Upn:       oauthConfig["upn"],
		},
	}
	return OAuthInfo
}
