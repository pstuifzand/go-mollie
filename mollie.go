/*
mollie.go - connect to the Mollie iDEAL API
Copyright (c) 2013 Peter Stuifzand

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/
// Package mollie helps you connect your program to the Mollie iDEAL API.
package mollie

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

type Mollie struct {
	baseurl    *url.URL
	partnerId  int
	testmode   bool
	profileKey string
}

type BankResponse struct {
	XMLName xml.Name `xml:"response"`
	Banks   []Bank   `xml:"bank"`
}

type Bank struct {
	XMLName xml.Name `xml:"bank"`
	Id      int      `xml:"bank_id"`
	Name    string   `xml:"bank_name"`
}

type FetchRequest struct {
	Amount      int
	BankId      int
	Description string
	Reporturl   *url.URL
	Returnurl   *url.URL
}

type MollieResponse struct {
	XMLName xml.Name `xml:"response"`
	Order   Order    `xml:"order"`
}

type Order struct {
	TransactionId string   `xml:"transaction_id"`
	Amount        int      `xml:"amount"`
	Currency      string   `xml:"currency"`
	Payed         bool     `xml:"payed"`
	Consumer      Consumer `xml:"consumer"`
	URL           string
	Message       string `xml:"message"`
	Status        string `xml:"status"`
}

type Consumer struct {
	Name    string `xml:"consumerName"`
	Account string `xml:"consumerAccount"`
	City    string `xml:"consumerCity"`
}

func (resp *MollieResponse) IsSuccess() bool {
	return resp.Order.Status == "Success"
}

func (resp *MollieResponse) IsCheckedBefore() bool {
	return resp.Order.Status == "CheckedBefore"
}

func (resp *MollieResponse) IsFailure() bool {
	return resp.Order.Status == "Failure"
}

func (resp *MollieResponse) IsExpired() bool {
	return resp.Order.Status == "Expired"
}

func (resp *MollieResponse) IsCancelled() bool {
	return resp.Order.Status == "Cancelled"
}

/*
NewMollie creates the main Mollie struct.

partnerId is the partnerId that you got from Mollie. If testmode is true the
requests will be sent in testmode.
*/
func NewMollie(partnerId int, testmode bool) (*Mollie, error) {
	baseurl, err := url.Parse("https://secure.mollie.nl/xml/ideal")
	if err != nil {
		return nil, err
	}
	if testmode {
		q := baseurl.Query()
		q.Set("testmode", "true")
		baseurl.RawQuery = q.Encode()
	}

	return &Mollie{baseurl: baseurl, partnerId: partnerId, testmode: testmode}, nil
}

/*
SetProfileKey allows you to set the optional profilekey.
*/
func (mollie *Mollie) SetProfileKey(key string) {
	mollie.profileKey = key
}

/*
BankList returns the banks that can be used right now.
*/
func (mollie *Mollie) BankList() (*BankResponse, error) {
	u := *mollie.baseurl
	q := u.Query()
	q.Set("a", "banklist")
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("StatusCode not 200, but %d", resp.StatusCode)
	}
	decoder := xml.NewDecoder(resp.Body)
	res := BankResponse{}
	err = decoder.Decode(&res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

/*
Fetch creates a new transaction. This is the command that should be called
after you recieve a bank_id from the client. Fetch requires multiple parameters
that can be set in the request parameter.

After Fetch you should redirect the client to URL in MollieResponse.
*/
func (mollie *Mollie) Fetch(request *FetchRequest) (*MollieResponse, error) {
	u := *mollie.baseurl
	q := u.Query()
	q.Set("a", "fetch")
	q.Set("partnerid", strconv.FormatInt(int64(mollie.partnerId), 10))
	if len(mollie.profileKey) > 0 {
		q.Set("profile_key", mollie.profileKey)
	}
	q.Set("amount", strconv.FormatInt(int64(request.Amount), 10))
	q.Set("bank_id", strconv.FormatInt(int64(request.BankId), 10))
	q.Set("description", request.Description)
	q.Set("reporturl", request.Reporturl.String())
	q.Set("returnurl", request.Returnurl.String())
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	decoder := xml.NewDecoder(resp.Body)
	res := MollieResponse{}
	err = decoder.Decode(&res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

/*
Check checks if the transaction is completed. It should be called when Mollie
calls you report url. Pass the transactionId of the transaction that you want
to check.
*/
func (mollie *Mollie) Check(transactionId string) (*MollieResponse, error) {
	u := *mollie.baseurl
	q := u.Query()
	q.Set("a", "check")
	q.Set("partnerid", strconv.FormatInt(int64(mollie.partnerId), 10))
	q.Set("transaction_id", transactionId)
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("\n%s\n", string(body))
	res := MollieResponse{}
	err = xml.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
