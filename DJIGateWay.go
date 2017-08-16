//接入DJI网关的封装

//
//example:
//server := "gateway-stg-alihz.aasky.net:9000"
//appID := "profession_iuav"
//appKey := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
//d, err := NewDJIGateWary(server, appID, appKey)
//fmt.Println(d.lastToken, err)

//do something
//token := "xxxxxxxxxxxxxxxxxxxxxxx"
//url := fmt.Sprintf("http://%s/gwapi/api/accounts/get_account_info_by_key?token=%s", d.Server, token)
//request, err := http.NewRequest("POST", url, nil)
//fmt.Println(request, err)

//requestServerID := "member_center"
//clientAddress := "127.0.0.1:8000"
//buf, err := d.DoRequest(request, requestServerID, clientAddress)
//

package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"errors"
)

type DJIGateWay struct {
	Server    string
	AppID     string
	AppKey    string
	refresh   time.Duration //refresh token secondes
	stop      chan bool
	lastToken DJIGateWayToken
	lastError error
	wg        sync.WaitGroup
	seq       uint64
}

//通过meta_key 获取token
type ItemInfo struct {
	Item string `json:"token"`
}

type TokenInfo struct {
	Status     int        `json:"status"`
	Status_msg string     `json:"status_msg"`
	Items      []ItemInfo `json:"items"`
}

//通过token获取账户信息
type DJIGateWayToken struct {
	AppID       string
	AccessToken string
	ExpireTime  int64
}

type RemoteError struct {
	Host string
	Err  error
}

func (e *RemoteError) Error() string {
	return fmt.Sprintf("%s-%s", e.Host, e.Err)
}

func NewDJIGateWay(server, appid, appkey string) (*DJIGateWay, error) {
	d := &DJIGateWay{
		Server:  server,
		AppID:   appid,
		AppKey:  appkey,
		refresh: 60 * 5, //default 5 min to refresh
		stop:    make(chan bool, 1),
	}
	d.getToken()
	if d.lastError == nil {
		go d.refreshToken()
	}
	return d, d.lastError
}

func (d *DJIGateWay) Close() {
	close(d.stop)
	d.wg.Wait()
}

func (d *DJIGateWay) getChallengeCode() (string, error) {
	requestURL := fmt.Sprintf("http://%s/api/token/challengeCode?appId=%s", d.Server, d.AppID)
	resp, err := httpGet(requestURL, nil)
	var code string
	if err != nil {
		d.lastError = err
	} else {
		defer resp.Body.Close()
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(resp.Body)
		code = string(buf.Bytes())
		d.lastError = err
	}
	return code, d.lastError
}

func (d *DJIGateWay) refreshToken() {
	d.wg.Add(1)
	defer d.wg.Done()
	for {
		select {
		case <-d.stop:
			return
		case <-time.After(time.Second * d.refresh):
			d.getToken()
		}
	}
}

func (d *DJIGateWay) getToken() {
	if code, err := d.getChallengeCode(); err == nil {
		code = strings.TrimSpace(code)
		mac := hmac.New(sha1.New, []byte(d.AppKey))
		mac.Write([]byte(code))
		key := mac.Sum(nil)
		sign := base64.StdEncoding.EncodeToString(key)
		sign = url.QueryEscape(sign)
		requestURL := fmt.Sprintf("http://%s/api/token?appId=%s&challengeCode=%s&signCode=%s", d.Server, d.AppID, code, sign)
		resp, err := httpGet(requestURL, nil)
		if err != nil {
			d.lastError = err
		} else {
			defer resp.Body.Close()
			buf := new(bytes.Buffer)
			_, err := buf.ReadFrom(resp.Body)
			json.Unmarshal(buf.Bytes(), &d.lastToken)
			d.lastError = err
		}
	}
}

func (d *DJIGateWay) GetAccessToken() (string, error) {
	return d.lastToken.AccessToken, d.lastError
}

func (d *DJIGateWay) getInvokeID() string {
	d.seq++
	return fmt.Sprintf("%d-%s-%d", time.Now().Unix(), d.AppID, d.seq)
}

func (d *DJIGateWay) DoRequest(r *http.Request, requestServerID, clientAddress string) (data []byte, err error) {
	//r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Connection", "Keep-Alive")
	//set gateway header
	r.Header.Set("userAgentIp-gw", clientAddress)
	r.Header.Set("consumerAppId-gw", d.AppID)
	r.Header.Set("providerAppId-gw", requestServerID)
	r.Header.Set("accessToken-gw", d.lastToken.AccessToken)
	r.Header.Set("invokeId-gw", d.getInvokeID())
	var resp *http.Response
	resp, err = http.DefaultClient.Do(r)
	if err != nil || resp.StatusCode != 200 {
		err = &RemoteError{fmt.Sprintf("request failed,[url:%s][httpcode:%d]", r.URL.String(), resp), err}
		return
	}
	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	return
}

func httpGet(url string, header http.Header) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range header {
		req.Header[k] = vs
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, &RemoteError{req.URL.Host, err}
	}
	if resp.StatusCode == 200 {
		return resp, nil
	}
	defer resp.Body.Close()
	err = &RemoteError{req.URL.Host, fmt.Errorf("request failed, [url:%s][httpcode:%d]", url, resp.StatusCode)}
	return nil, err
}

func (d *DJIGateWay) CheckToken(token string, clientAddress string) (UserInfoResult, error) {
	var info UserInfoResult

	if len(token) == 0 {
		return info, errors.New("token is empty")
	}

	uri := fmt.Sprintf("http://%s/gwapi/api/accounts/get_account_info_by_key?token=%s", d.Server, token)
	request, err := http.NewRequest("POST", uri, nil)
	if err == nil {
		var buf []byte

		buf, err = d.DoRequest(request, APP_MEMBER_CENTER, clientAddress)
		if err == nil {
			json.Unmarshal(buf, &info)
		}
	}

	return info, err
}

func (d *DJIGateWay) GetTokenByCookie(cookie string, addr string) (string, error) {
	var token TokenInfo

	uri := fmt.Sprintf("http://%s/gwapi/api/accounts/get_token_by_meta_key?meta_key=%s", service.djiGateWay.Server, cookie)
	request, err := http.NewRequest("POST", uri, nil)
	if err != nil {
		fmt.Println("new request error", err)
		return "", err
	}

	buf, err := d.DoRequest(request, APP_MEMBER_CENTER, addr)
	if err != nil {
		fmt.Println("get token by cookie err", err)
		return "", err
	}

	json.Unmarshal(buf, &token)
	if err != nil || token.Status != 0 || len(token.Items) == 0 {
		fmt.Println("app auth failed", cookie, err, token)
		return "", errors.New("get token failed")
	}

	return token.Items[0].Item, nil
}
