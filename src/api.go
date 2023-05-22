package main

import (
	"encoding/json"
	"net/http"

	"imuslab.com/zoraxy/mod/auth"
	"imuslab.com/zoraxy/mod/netstat"
	"imuslab.com/zoraxy/mod/utils"
)

/*
	API.go

	This file contains all the API called by the web management interface

*/

var requireAuth = true

func initAPIs() {

	authRouter := auth.NewManagedHTTPRouter(auth.RouterOption{
		AuthAgent:   authAgent,
		RequireAuth: requireAuth,
		DeniedHandler: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "401 - Unauthorized", http.StatusUnauthorized)
		},
	})

	//Register the standard web services urls
	fs := http.FileServer(http.FS(webres))
	if development {
		fs = http.FileServer(http.Dir("web/"))
	}
	//Add a layer of middleware for advance  control
	advHandler := FSHandler(fs)
	http.Handle("/", advHandler)

	//Authentication APIs
	registerAuthAPIs(requireAuth)

	//Reverse proxy
	authRouter.HandleFunc("/api/proxy/enable", ReverseProxyHandleOnOff)
	authRouter.HandleFunc("/api/proxy/add", ReverseProxyHandleAddEndpoint)
	authRouter.HandleFunc("/api/proxy/status", ReverseProxyStatus)
	authRouter.HandleFunc("/api/proxy/list", ReverseProxyList)
	authRouter.HandleFunc("/api/proxy/del", DeleteProxyEndpoint)
	authRouter.HandleFunc("/api/proxy/tlscheck", HandleCheckSiteSupportTLS)
	authRouter.HandleFunc("/api/proxy/setIncoming", HandleIncomingPortSet)
	authRouter.HandleFunc("/api/proxy/useHttpsRedirect", HandleUpdateHttpsRedirect)
	authRouter.HandleFunc("/api/proxy/requestIsProxied", HandleManagementProxyCheck)

	//TLS / SSL config
	authRouter.HandleFunc("/api/cert/tls", handleToggleTLSProxy)
	authRouter.HandleFunc("/api/cert/upload", handleCertUpload)
	authRouter.HandleFunc("/api/cert/list", handleListCertificate)
	authRouter.HandleFunc("/api/cert/checkDefault", handleDefaultCertCheck)
	authRouter.HandleFunc("/api/cert/delete", handleCertRemove)

	//Redirection config
	authRouter.HandleFunc("/api/redirect/list", handleListRedirectionRules)
	authRouter.HandleFunc("/api/redirect/add", handleAddRedirectionRule)
	authRouter.HandleFunc("/api/redirect/delete", handleDeleteRedirectionRule)

	//Blacklist APIs
	authRouter.HandleFunc("/api/blacklist/list", handleListBlacklisted)
	authRouter.HandleFunc("/api/blacklist/country/add", handleCountryBlacklistAdd)
	authRouter.HandleFunc("/api/blacklist/country/remove", handleCountryBlacklistRemove)
	authRouter.HandleFunc("/api/blacklist/ip/add", handleIpBlacklistAdd)
	authRouter.HandleFunc("/api/blacklist/ip/remove", handleIpBlacklistRemove)
	authRouter.HandleFunc("/api/blacklist/enable", handleBlacklistEnable)

	//Statistic & uptime monitoring API
	authRouter.HandleFunc("/api/stats/summary", statisticCollector.HandleTodayStatLoad)
	authRouter.HandleFunc("/api/stats/countries", HandleCountryDistrSummary)
	authRouter.HandleFunc("/api/stats/netstat", netstat.HandleGetNetworkInterfaceStats)
	authRouter.HandleFunc("/api/stats/netstatgraph", netstatBuffers.HandleGetBufferedNetworkInterfaceStats)
	authRouter.HandleFunc("/api/stats/listnic", netstat.HandleListNetworkInterfaces)
	authRouter.HandleFunc("/api/utm/list", HandleUptimeMonitorListing)

	//Global Area Network APIs
	authRouter.HandleFunc("/api/gan/network/info", ganManager.HandleGetNodeID)
	authRouter.HandleFunc("/api/gan/network/add", ganManager.HandleAddNetwork)
	authRouter.HandleFunc("/api/gan/network/remove", ganManager.HandleRemoveNetwork)
	authRouter.HandleFunc("/api/gan/network/list", ganManager.HandleListNetwork)
	authRouter.HandleFunc("/api/gan/network/name", ganManager.HandleNetworkNaming)
	//authRouter.HandleFunc("/api/gan/network/detail", ganManager.HandleNetworkDetails)
	authRouter.HandleFunc("/api/gan/network/setRange", ganManager.HandleSetRanges)
	authRouter.HandleFunc("/api/gan/members/list", ganManager.HandleMemberList)
	authRouter.HandleFunc("/api/gan/members/ip", ganManager.HandleMemberIP)
	authRouter.HandleFunc("/api/gan/members/name", ganManager.HandleMemberNaming)
	authRouter.HandleFunc("/api/gan/members/authorize", ganManager.HandleMemberAuthorization)
	authRouter.HandleFunc("/api/gan/members/delete", ganManager.HandleMemberDelete)

	//TCP Proxy
	authRouter.HandleFunc("/api/tcpprox/config/add", tcpProxyManager.HandleAddProxyConfig)
	authRouter.HandleFunc("/api/tcpprox/config/edit", tcpProxyManager.HandleEditProxyConfigs)
	authRouter.HandleFunc("/api/tcpprox/config/list", tcpProxyManager.HandleListConfigs)
	authRouter.HandleFunc("/api/tcpprox/config/status", tcpProxyManager.HandleGetProxyStatus)
	authRouter.HandleFunc("/api/tcpprox/config/validate", tcpProxyManager.HandleConfigValidate)

	//mDNS APIs
	authRouter.HandleFunc("/api/mdns/list", HandleMdnsListing)
	authRouter.HandleFunc("/api/mdns/discover", HandleMdnsScanning)

	//Zoraxy Analytic
	authRouter.HandleFunc("/api/analytic/list", AnalyticLoader.HandleSummaryList)
	authRouter.HandleFunc("/api/analytic/load", AnalyticLoader.HandleLoadTargetDaySummary)
	authRouter.HandleFunc("/api/analytic/loadRange", AnalyticLoader.HandleLoadTargetRangeSummary)

	//Network utilities
	authRouter.HandleFunc("/api/tools/ipscan", HandleIpScan)
	authRouter.HandleFunc("/api/tools/webssh", HandleCreateProxySession)
	authRouter.HandleFunc("/api/tools/websshSupported", HandleWebSshSupportCheck)
	authRouter.HandleFunc("/api/tools/wol", HandleWakeOnLan)
	authRouter.HandleFunc("/api/tools/smtp/get", HandleSMTPGet)
	authRouter.HandleFunc("/api/tools/smtp/set", HandleSMTPSet)
	authRouter.HandleFunc("/api/tools/smtp/admin", HandleAdminEmailGet)
	authRouter.HandleFunc("/api/tools/smtp/test", HandleTestEmailSend)

	//Account Reset
	http.HandleFunc("/api/account/reset", HandleAdminAccountResetEmail)
	http.HandleFunc("/api/account/new", HandleNewPasswordSetup)

	//If you got APIs to add, append them here
}

// Function to renders Auth related APIs
func registerAuthAPIs(requireAuth bool) {
	//Auth APIs
	http.HandleFunc("/api/auth/login", authAgent.HandleLogin)
	http.HandleFunc("/api/auth/logout", authAgent.HandleLogout)
	http.HandleFunc("/api/auth/checkLogin", func(w http.ResponseWriter, r *http.Request) {
		if requireAuth {
			authAgent.CheckLogin(w, r)
		} else {
			utils.SendJSONResponse(w, "true")
		}
	})
	http.HandleFunc("/api/auth/username", func(w http.ResponseWriter, r *http.Request) {
		username, err := authAgent.GetUserName(w, r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		js, _ := json.Marshal(username)
		utils.SendJSONResponse(w, string(js))
	})
	http.HandleFunc("/api/auth/userCount", func(w http.ResponseWriter, r *http.Request) {
		uc := authAgent.GetUserCounts()
		js, _ := json.Marshal(uc)
		utils.SendJSONResponse(w, string(js))
	})
	http.HandleFunc("/api/auth/register", func(w http.ResponseWriter, r *http.Request) {
		if authAgent.GetUserCounts() == 0 {
			//Allow register root admin
			authAgent.HandleRegisterWithoutEmail(w, r, func(username, reserved string) {

			})
		} else {
			//This function is disabled
			utils.SendErrorResponse(w, "Root management account already exists")
		}
	})
	http.HandleFunc("/api/auth/changePassword", func(w http.ResponseWriter, r *http.Request) {
		username, err := authAgent.GetUserName(w, r)
		if err != nil {
			http.Error(w, "401 - Unauthorized", http.StatusUnauthorized)
			return
		}

		oldPassword, err := utils.PostPara(r, "oldPassword")
		if err != nil {
			utils.SendErrorResponse(w, "empty current password")
			return
		}
		newPassword, err := utils.PostPara(r, "newPassword")
		if err != nil {
			utils.SendErrorResponse(w, "empty new password")
			return
		}
		confirmPassword, _ := utils.PostPara(r, "confirmPassword")

		if newPassword != confirmPassword {
			utils.SendErrorResponse(w, "confirm password not match")
			return
		}

		//Check if the old password correct
		oldPasswordCorrect, _ := authAgent.ValidateUsernameAndPasswordWithReason(username, oldPassword)
		if !oldPasswordCorrect {
			utils.SendErrorResponse(w, "Invalid current password given")
			return
		}

		//Change the password of the root user
		authAgent.UnregisterUser(username)
		authAgent.CreateUserAccount(username, newPassword, "")
	})

}
