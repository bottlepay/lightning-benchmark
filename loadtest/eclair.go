package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type eclairConnection struct {
	client     *http.Client
	host, pass string
}

func getEclairConnection(cfg *eclairConfig) (*eclairConnection, error) {
	client := http.Client{}

	conn := &eclairConnection{
		client: &client,
		host:   cfg.RpcHost,
		pass:   cfg.Password,
	}

	logger := log.With("host", cfg.RpcHost)

	logger.Infow("Attempting to connect to eclair")
	for {
		info, err := conn.GetInfo()
		if err == nil {
			logger.Infow("Connected to eclair", "key", info.key)
			break
		}

		time.Sleep(time.Second)
	}

	return conn, nil
}

func (l *eclairConnection) Close() {
}

func (l *eclairConnection) call(method string, parameters map[string]string) (
	[]byte, error) {

	uri := fmt.Sprintf("http://%v/%v", l.host, method)
	data := url.Values{}
	for k, v := range parameters {
		data[k] = []string{v}
	}
	body := strings.NewReader(data.Encode())

	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("", l.pass)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func (l *eclairConnection) GetInfo() (*info, error) {
	respBytes, err := l.call("getinfo", map[string]string{})
	if err != nil {
		return nil, err
	}

	var resp struct {
		NodeId string
	}
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		log.Errorw("json deserialize error",
			"err", err, "data", string(respBytes))

		return nil, err
	}

	return &info{
		key:    resp.NodeId,
		synced: true,
	}, nil
}

func (l *eclairConnection) Connect(key, host string) error {
	uri := fmt.Sprintf("%v@%v", key, host)
	_, err := l.call("connect", map[string]string{"uri": uri})
	return err
}

func (l *eclairConnection) NewAddress() (string, error) {
	respBytes, err := l.call("getnewaddress", map[string]string{})
	if err != nil {
		return "", err
	}

	var addr string
	err = json.Unmarshal(respBytes, &addr)
	if err != nil {
		log.Errorw("json deserialize error",
			"err", err, "data", string(respBytes))

		return "", err
	}
	return addr, nil
}

func (l *eclairConnection) OpenChannel(peerKey string, amtSat int64) error {
	_, err := l.call("open",
		map[string]string{
			"nodeId":          peerKey,
			"fundingSatoshis": strconv.FormatInt(amtSat, 10),
		})
	return err
}

func (l *eclairConnection) ActiveChannels() (int, error) {
	respBytes, err := l.call("channels", map[string]string{})
	if err != nil {
		return 0, err
	}

	var channels []struct {
		State string
	}
	err = json.Unmarshal(respBytes, &channels)
	if err != nil {
		log.Errorw("json deserialize error",
			"err", err, "data", string(respBytes))

		return 0, err
	}

	var activeCount int
	for _, ch := range channels {
		if ch.State == "NORMAL" {
			activeCount++
		}
	}

	return activeCount, nil
}

func (l *eclairConnection) AddInvoice(amtMsat int64) (string, error) {
	for {
		invoice, err := l.addInvoice(amtMsat)
		if err == nil {
			return invoice, nil
		}

		log.Warnw("Invoice generation failed", "err", err)
		time.Sleep(time.Second)
	}
}

func (l *eclairConnection) addInvoice(amtMsat int64) (string, error) {
	respBytes, err := l.call("createinvoice", map[string]string{
		"amountMsat":  strconv.FormatInt(amtMsat, 10),
		"description": "test",
	})
	if err != nil {
		return "", err
	}

	var respJson struct {
		Serialized string
	}
	err = json.Unmarshal(respBytes, &respJson)
	if err != nil {
		log.Errorw("json deserialize error",
			"err", err, "data", string(respBytes))

		return "", err
	}

	invoice := respJson.Serialized
	if invoice == "" {
		return "", errors.New("no invoice returned")
	}

	return invoice, nil
}

func (l *eclairConnection) SendPayment(invoice string) error {
	respBytes, err := l.call("payinvoice", map[string]string{
		"invoice":     invoice,
		"description": "test",
	})
	if err != nil {
		return err
	}

	var id string
	err = json.Unmarshal(respBytes, &id)
	if err != nil {
		log.Errorw("json deserialize error",
			"err", err, "data", string(respBytes), "invoice", invoice)

		return err
	}

	return l.waitForSent(id)
}

func (l *eclairConnection) waitForSent(id string) error {
	for {
		respBytes, err := l.call("getsentinfo", map[string]string{
			"id": id,
		})
		if err != nil {
			return err
		}

		var resp []struct {
			Status struct {
				Type string
			}
		}
		err = json.Unmarshal(respBytes, &resp)
		if err != nil {
			log.Errorw("json deserialize error",
				"err", err, "data", string(respBytes))

			return err
		}

		for _, attempt := range resp {
			if attempt.Status.Type == "sent" {
				return nil
			}
		}

		time.Sleep(time.Second)
	}
}

func (l *eclairConnection) SendKeysend(destination string, amtMsat int64) error {
	return errors.New("not implemented")
}

func (l *eclairConnection) HasFunds() error {
	for {
		respBytes, err := l.call("onchainbalance", map[string]string{})
		if err != nil {
			return err
		}

		var resp struct {
			Confirmed int
		}
		err = json.Unmarshal(respBytes, &resp)
		if err != nil {
			log.Errorw("json deserialize error",
				"err", err, "data", string(respBytes))

			return err
		}
		if resp.Confirmed != 0 {
			return nil
		}
	}
}
