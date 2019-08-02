package nickserver

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/boreq/starlight-nick-server/data"
	"github.com/boreq/starlight/network/node"
	"github.com/pkg/errors"
)

const timeout = 10 * time.Second

const putUrl = "nicks"
const getUrl = "nicks/<id>"
const getByNickUrl = "ids/<nick>"

func NewNickServerClient(baseUrl string, iden *node.Identity) (*NickServerClient, error) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		return nil, errors.Wrap(err, "invalid base url")
	}

	client := &http.Client{
		Timeout: timeout,
	}

	rv := &NickServerClient{
		baseUrl: u,
		client:  client,
		iden:    iden,
	}
	return rv, nil
}

type NickServerClient struct {
	baseUrl *url.URL
	client  *http.Client
	iden    *node.Identity
}

func (n *NickServerClient) ValidateNick(nick string) error {
	return data.ValidateNick(nick)
}

func (n *NickServerClient) Put(nick string) error {
	nickData, err := n.createNickData(nick)
	if err != nil {
		return errors.Wrap(err, "nick data creation failed")
	}

	b, err := json.Marshal(nickData)
	if err != nil {
		return errors.Wrap(err, "encoding as json failed")
	}

	buf := &bytes.Buffer{}
	buf.Write(b)

	u, err := n.getUrl(putUrl)
	if err != nil {
		return errors.Wrap(err, "could not get url")
	}

	request, err := http.NewRequest(http.MethodPut, u, buf)
	if err != nil {
		return errors.Wrap(err, "request creation failed")
	}

	response, err := n.client.Do(request)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}

	if response.StatusCode != 200 {
		return errors.New("server returned an error")
	}

	return nil
}

func (n *NickServerClient) createNickData(nick string) (*data.NickData, error) {
	pubKeyBytes, err := n.iden.PubKey.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "could not get public key bytes")
	}
	nickData := &data.NickData{
		Id:        n.iden.Id,
		Nick:      nick,
		Time:      time.Now(),
		PublicKey: pubKeyBytes,
	}
	dataToSign := nickData.GetDataToSign()
	signature, err := n.iden.PrivKey.Sign(dataToSign, data.SigningHash)
	if err != nil {
		return nil, errors.Wrap(err, "signature creation failed")
	}
	nickData.Signature = signature
	if err := nickData.Validate(); err != nil {
		return nil, errors.Wrap(err, "created nick data is not valid")
	}
	return nickData, nil
}

func (n *NickServerClient) Get(id node.ID) (string, error) {
	url, err := n.getUrl(strings.ReplaceAll(getUrl, "<id>", id.String()))
	if err != nil {
		return "", errors.Wrap(err, "could not get url")
	}

	response, err := n.client.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "request failed")
	}

	if response.StatusCode != 200 {
		return "", errors.New("server returned an error")
	}

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", errors.Wrap(err, "reading the response failed")
	}

	nickData := &data.NickData{}
	if err := json.Unmarshal(b, nickData); err != nil {
		return "", errors.Wrap(err, "decoding json failed")
	}

	if !node.CompareId(id, nickData.Id) {
		return "", errors.New("server returned data for a wrong node")
	}

	if err := nickData.Validate(); err != nil {
		return "", errors.Wrap(err, "server returned invalid data")
	}

	return nickData.Nick, nil
}

func (n *NickServerClient) GetByNick(nick string) (node.ID, error) {
	url, err := n.getUrl(strings.ReplaceAll(getByNickUrl, "<nick>", nick))
	if err != nil {
		return nil, errors.Wrap(err, "could not get url")
	}

	response, err := n.client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}

	if response.StatusCode != 200 {
		return nil, errors.New("server returned an error")
	}

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading the response failed")
	}

	nickData := &data.NickData{}
	if err := json.Unmarshal(b, nickData); err != nil {
		return nil, errors.Wrap(err, "decoding json failed")
	}

	if nick != nickData.Nick {
		return nil, errors.New("server returned data for a wrong node")
	}

	if err := nickData.Validate(); err != nil {
		return nil, errors.Wrap(err, "server returned invalid data")
	}

	return nickData.Id, nil
}

func (n *NickServerClient) getUrl(s string) (string, error) {
	u, err := url.Parse(n.baseUrl.String())
	if err != nil {
		return "", errors.Wrap(err, "could not parse base url")
	}
	u.Path = path.Join(u.Path, s)
	return u.String(), nil
}
