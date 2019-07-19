package client

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/cortexproject/cortex/pkg/ruler/store"
	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/pkg/rulefmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	ErrNoConfig         = errors.New("No config exists for this user")
	ErrResourceNotFound = errors.New("requested resource not found")
)

// Config is used to configure a Ruler Client
type Config struct {
	Key     string `yaml:"key"`
	Address string `yaml:"address"`
	ID      string `yaml:"id"`
}

// RulerClient is used to get and load rules into a cortex ruler
type RulerClient struct {
	key      string
	id       string
	endpoint *url.URL
	client   http.Client
}

// New returns a new Client
func New(cfg Config) (*RulerClient, error) {
	endpoint, err := url.Parse(cfg.Address)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"address": cfg.Address,
		"id":      cfg.ID,
	}).Debugln("New ruler client created")

	return &RulerClient{
		key:      cfg.Key,
		id:       cfg.ID,
		endpoint: endpoint,
		client:   http.Client{},
	}, nil
}

func (r *RulerClient) doRequest(path, method string, payload []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, r.endpoint.String()+path, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	if r.key != "" {
		req.SetBasicAuth(r.id, r.key)
	}

	req.Header.Add("X-Scope-OrgID", r.id)

	log.WithFields(log.Fields{
		"url": req.URL.String(),
	}).Debugln("sending request to ruler api")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}

	err = checkResponse(resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// checkResponse checks the API response for errors
func checkResponse(r *http.Response) error {
	log.WithFields(log.Fields{
		"status": r.Status,
	}).Debugln("checking response")
	if 200 <= r.StatusCode && r.StatusCode <= 299 {
		return nil
	}

	var msg string
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		msg = fmt.Sprintf("unable to decode body, %s", err.Error())
	} else {
		msg = fmt.Sprintf("request failed with response body %v", string(data))
	}

	if r.StatusCode == http.StatusNotFound {
		log.WithFields(log.Fields{
			"status": r.Status,
			"msg":    msg,
		}).Debugln("resource not found")
		return ErrResourceNotFound
	}

	log.WithFields(log.Fields{
		"status": r.Status,
		"msg":    msg,
	}).Errorln("requests failed")

	return errors.New("failed request to the ruler api")
}

// CreateRuleGroup creates a new rule group
func (r *RulerClient) CreateRuleGroup(ctx context.Context, namespace string, rg rulefmt.RuleGroup) error {
	payload, err := yaml.Marshal(&rg)
	if err != nil {
		return err
	}

	res, err := r.doRequest("/api/prom/rules/"+namespace, "POST", payload)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	err = checkResponse(res)
	if err != nil {
		return err
	}

	return nil
}

// DeleteRuleGroup creates a new rule group
func (r *RulerClient) DeleteRuleGroup(ctx context.Context, namespace, groupName string) error {
	res, err := r.doRequest("/api/prom/rules/"+namespace, "DELETE", nil)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	err = checkResponse(res)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	if res.StatusCode%2 > 0 {
		return fmt.Errorf("error occured, %v", string(body))
	}

	return nil
}

// GetRuleGroup retrieves a rule group
func (r *RulerClient) GetRuleGroup(ctx context.Context, namespace, groupName string) (*rulefmt.RuleGroup, error) {
	res, err := r.doRequest(fmt.Sprintf("/api/prom/rules/%s/%s", namespace, groupName), "GET", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform request")
	}

	defer res.Body.Close()
	err = checkResponse(res)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	rg := rulefmt.RuleGroup{}
	err = yaml.Unmarshal(body, &rg)
	if err != nil {
		log.WithFields(log.Fields{
			"body": string(body),
		}).Debugln("failed to unmarshal rule group from response")

		return nil, errors.Wrap(err, "unable to unmarshal response")
	}

	return &rg, nil
}

// ListRules retrieves a rule group
func (r *RulerClient) ListRules(ctx context.Context, namespace string) (map[string]store.RuleNamespace, error) {
	path := "/api/prom/rules"
	if namespace != "" {
		path = path + "/" + namespace
	}

	res, err := r.doRequest(path, "GET", nil)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	err = checkResponse(res)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	ruleSet := map[string]store.RuleNamespace{}
	err = yaml.Unmarshal(body, &ruleSet)
	if err != nil {
		return nil, err
	}

	return ruleSet, nil
}
