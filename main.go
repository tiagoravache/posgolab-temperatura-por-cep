package main

import (
	"encoding/json"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"unicode"
)

type ViaCep struct {
	Localidade string `json:"localidade"`
	Erro       bool   `json:"erro"`
}

type WeatherApi struct {
	Current struct {
		TempC float64 `json:"temp_c"`
		TempF float64 `json:"temp_f"`
		TempK float64 `json:"temp_k"`
	} `json:"current"`
}

func main() {
	http.HandleFunc("/", SearchCepHandler)
	http.ListenAndServe(":8080", nil)
}

func SearchCepHandler(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	cepParam := r.URL.Query().Get("cep")
	if cepParam == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("cep parameter is required"))
		return
	}

	validate := regexp.MustCompile(`^[0-9]{8}$`)
	if !validate.MatchString(cepParam) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("invalid zipcode"))
		return
	}

	cep, err := SearchCep(cepParam)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorStr := err.Error()
		w.Write([]byte("error while searching for cep: " + errorStr))
		return
	}

	if cep.Erro {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("can not find zipcode"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	weather, err := SearchTemperature(cep.Localidade)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorStr := err.Error()
		w.Write([]byte("error while searching for temperature: " + errorStr))
		return
	}
	if weather != nil {
		json.NewEncoder(w).Encode(weather.Current)
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("can not find temperature"))
	}
}

func SearchCep(cep string) (*ViaCep, error) {
	req, err := http.Get("http://viacep.com.br/ws/" + cep + "/json/")
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()

	res, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	var data ViaCep
	err = json.Unmarshal(res, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func SearchTemperature(city string) (*WeatherApi, error) {
	req, err := http.Get("http://api.weatherapi.com/v1/current.json?key=<YOUR_KEY>&aqi=no&q=" + removeDiacriticsAndEncodeCityName(city))

	if err != nil {
		return nil, err
	}
	defer req.Body.Close()

	res, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	var data WeatherApi
	err = json.Unmarshal(res, &data)
	if err != nil {
		return nil, err
	}

	data.Current.TempF = data.Current.TempC*1.8 + 32
	data.Current.TempK = data.Current.TempC + 273

	return &data, nil
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}

func removeDiacriticsAndEncodeCityName(s string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, _ := transform.String(t, s)
	result = url.QueryEscape(result)
	println(result)
	return result
}
