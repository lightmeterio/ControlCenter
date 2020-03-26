#!/usr/bin/env python3

def parse_args():
    import argparse
    parser = argparse.ArgumentParser(description="Lightmeter ControlCenter Simple CLI Dashboard")
    parser.add_argument("host", help="URL for Lightmeter ControlCenter")
    return parser.parse_args()

args = parse_args()

lightmeter_api = f"{args.host}/api/"

def req_json(method):
    import requests

    r = requests.get(lightmeter_api + "/" + method)

    if r.status_code != 200 or r.headers['Content-Type'] != "application/json":
        raise Exception("Error Querying Lightmeter")

    return r.json()

def build_delivery_status():
    s = req_json("deliveryStatus")

    def f(status):
        for p in s:
            if p['Status'] == status:
                return p['Value']

        raise Exception(r"Invalid status: {status}")
    
    return f

def list_domains_with_counts(title, method):
    l = req_json(method)

    print(f"\n{title}:\n")
    for e in l:
        print(f"{e['Domain']} {e['Count']}")
    
del_status = build_delivery_status()

print("Summary:\n")

print(f"{del_status('sent')} Sent")
print(f"{del_status('bounced')} Bounced")
print(f"{del_status('deferred')} Deferred")

list_domains_with_counts("Busiest Domains", "topBusiestDomains")
list_domains_with_counts("Top Deferred Domains", "topDeferredDomains")
list_domains_with_counts("Top Bounced Domains", "topBouncedDomains")
