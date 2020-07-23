LinInfoSec
==========

LinInfoSec is a CVE monitoring tool exposing a REST API for monitoring CVE feeds for publications affecting software configurations.

It exposes the following API:

- '/monitor/add' accepting HTTP POST requests, to add a configuration of software to monitor 
- '/monitor/remove' accepting HTTP POST requests, to remove a configuration of software to monitor 
- '/searchCPE' accepting HTTP GET requests, to search among the CPE dictionnary

It fetches regularly the CVE feeds from NVD. It then compares new CVE publications with the configurations being monitored.
When a vulnerability is detected a HTTP POST request is sent to a configured endpoint with the list of vulnerabilities in JSON format:

```json
[
	{
		"configuration" : {
			"type": "string"
		},
		"cve" :  {
			"type": "CVE_JSON_4.0_min_1.1.schema"
		}
	}
]
```
