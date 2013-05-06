// Copyright 2013 Arne Roomann-Kurrik
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

func BaseHandler(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "text/html;charset=utf-8")
	fmt.Fprintf(rw, "<a href=\"/signin\">Sign in</a>")
}

func SignInHandler(rw http.ResponseWriter, req *http.Request) {
	var (
		url       string
		err       error
		sessionID string
	)
	httpClient := new(http.Client)
	userConfig := &oauth1a.UserConfig{}
	if err = userConfig.GetRequestToken(service, httpClient); err != nil {
		log.Printf("Could not get request token: %v", err)
		http.Error(rw, "Problem getting the request token", 500)
		return
	}
	if url, err = userConfig.GetAuthorizeURL(service); err != nil {
		log.Printf("Could not get authorization URL: %v", err)
		http.Error(rw, "Problem getting the authorization URL", 500)
		return
	}
	log.Printf("Redirecting user to %v\n", url)
	sessionID = NewSessionID()
	log.Printf("Starting session %v\n", sessionID)
	sessions[sessionID] = userConfig
	http.SetCookie(rw, SessionStartCookie(sessionID))
	http.Redirect(rw, req, url, 302)
}

func CallbackHandler(rw http.ResponseWriter, req *http.Request) {
	var (
		err        error
		token      string
		verifier   string
		sessionID  string
		userConfig *oauth1a.UserConfig
		ok         bool
	)
	log.Printf("Callback hit. %v current sessions.\n", len(sessions))
	if sessionID, err = GetSessionID(req); err != nil {
		log.Printf("Got a callback with no session id: %v\n", err)
		http.Error(rw, "No session found", 400)
		return
	}
	if userConfig, ok = sessions[sessionID]; !ok {
		log.Printf("Could not find user config in sesions storage.")
		http.Error(rw, "Invalid session", 400)
		return
	}
	if token, verifier, err = userConfig.ParseAuthorize(req, service); err != nil {
		log.Printf("Could not parse authorization: %v", err)
		http.Error(rw, "Problem parsing authorization", 500)
		return
	}
	httpClient := new(http.Client)
	if err = userConfig.GetAccessToken(token, verifier, service, httpClient); err != nil {
		log.Printf("Error getting access token: %v", err)
		http.Error(rw, "Problem getting an access token", 500)
		return
	}
	log.Printf("Ending session %v.\n", sessionID)
	delete(sessions, sessionID)
	http.SetCookie(rw, SessionEndCookie())
	rw.Header().Set("Content-Type", "text/html;charset=utf-8")
	fmt.Fprintf(rw, "<pre>")
	fmt.Fprintf(rw, "Access Token: %v\n", userConfig.AccessTokenKey)
	fmt.Fprintf(rw, "Token Secret: %v\n", userConfig.AccessTokenSecret)
	fmt.Fprintf(rw, "Screen Name:  %v\n", userConfig.AccessValues.Get("screen_name"))
	fmt.Fprintf(rw, "User ID:      %v\n", userConfig.AccessValues.Get("user_id"))
	fmt.Fprintf(rw, "</pre>")
	fmt.Fprintf(rw, "<a href=\"/signin\">Sign in again</a>")
}

type Settings struct {
	Key  string
	Sec  string
	Port int
}

func init() {
	sessions = map[string]*oauth1a.UserConfig{}
	settings = &Settings{}
	flag.IntVar(&settings.Port, "port", 10000, "Port to run on")
	flag.StringVar(&settings.Key, "key", "", "Consumer key of your app")
	flag.StringVar(&settings.Sec, "secret", "", "Consumer secret of your app")
	flag.Parse()
	if settings.Key == "" || settings.Sec == "" {
		fmt.Fprintf(os.Stderr, "You must specify a consumer key and secret.\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	service = &oauth1a.Service{
		RequestURL:   "https://api.twitter.com/oauth/request_token",
		AuthorizeURL: "https://api.twitter.com/oauth/authorize",
		AccessURL:    "https://api.twitter.com/oauth/access_token",
		ClientConfig: &oauth1a.ClientConfig{
			ConsumerKey:    settings.Key,
			ConsumerSecret: settings.Sec,
			CallbackURL:    "http://localhost:10000/callback/",
		},
		Signer: new(oauth1a.HmacSha1Signer),
	}

	http.HandleFunc("/", BaseHandler)
	http.HandleFunc("/signin/", SignInHandler)
	http.HandleFunc("/callback/", CallbackHandler)
	log.Printf("Visit http://localhost:%v in your browser\n", settings.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", settings.Port), nil))
}
