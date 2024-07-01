package types

import "encoding/xml"

type CustomDomainMetadata struct {
	XMLName xml.Name `xml:"metadata"`
	App     App      `xml:"app"`
}

type App struct {
	XMLName xml.Name `xml:"app"`
	NS      string   `xml:"xmlns,attr"`
	From    string   `xml:"from,attr"`
	Owner   AppOwner `xml:"owner"`
	Name    AppName  `xml:"name"`
	IP      AppIP    `xml:"ip"`
	ID      AppID    `xml:"id"`
}

type AppOwner struct {
	XMLName  xml.Name `xml:"owner"`
	UserID   string   `xml:"id,attr"`
	UserName string   `xml:",chardata"`
}

type AppName struct {
	XMLName xml.Name `xml:"name"`
	Name    string   `xml:",chardata"`
}

type AppIP struct {
	XMLName xml.Name `xml:"ip"`
	IP      string   `xml:",chardata"`
}

type AppID struct {
	XMLName xml.Name `xml:"id"`
	SID     string   `xml:"sid,attr"`
	ID      string   `xml:",chardata"`
}
