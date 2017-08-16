package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type ServiceScan struct {
	ServiceScan string `xml:"servicescan"`
	Hosts       []Host `xml:"host"`
}

type Host struct {
	Address string `xml:"address,attr"`
	Ports   []Port `xml:"port"`
}

type Port struct {
	Number   string `xml:"number,attr"`
	Protocol string `xml:"protocol,attr"`
	State    string `xml:"state,attr"`
	Service  string `xml:"description,attr"`
}

func displayPort(checklist *bool, port *Port) {
	if *checklist {
		fmt.Printf(" [ ]")
	}
	fmt.Printf(" %s %s %s\n", port.Protocol, port.Number, port.Service)
}

func main() {
	ports := flag.String("p", "all", "Filter by ports")
	checklist := flag.Bool("c", false, "Display with checklist")
	inputFile := flag.String("i", "", "Servicescan xml file")

	fmt.Printf("ports: %v\n", *ports)
	fmt.Printf("checklist: %v\n", *checklist)
	fmt.Printf("inputFile: %v\n", *inputFile)

	flag.Parse()

	if *inputFile == "" {
		flag.Usage()
		return
	}

	xmlFile, err := os.Open(*inputFile)
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
	}
	defer xmlFile.Close()

	data, _ := ioutil.ReadAll(xmlFile)

	var v ServiceScan

	err = xml.Unmarshal([]byte(data), &v)
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}

	noFilter := true
	var filtering_ports []string
	if *ports != "all" {
		noFilter = false
		filtering_ports = strings.Split(*ports, ",")
	}

	for _, host := range v.Hosts {
		showedAddr := false
		if len(host.Ports) == 0 {
			continue
		}

		for _, port := range host.Ports {
			if noFilter && !showedAddr {
				fmt.Printf("%s\n", host.Address)
				showedAddr = true
			}

			if port.State != "open" {
				continue
			}

			if !noFilter {
				for _, filteringPort := range filtering_ports {
					if filteringPort == port.Number {
						if !showedAddr {
							fmt.Printf("%s\n", host.Address)
							showedAddr = true
						}
						displayPort(checklist, &port)
					}
				}
			} else {
				displayPort(checklist, &port)
			}
		}
		if !noFilter && showedAddr {
			fmt.Println()
		}
	}
}
