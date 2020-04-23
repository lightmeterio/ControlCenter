#!/usr/bin/env python3

def parse_args():
    import argparse
    parser = argparse.ArgumentParser(description="Lightmeter ControlCenter Simple CLI Dashboard")
    parser.add_argument("host", help="URL for Lightmeter ControlCenter")
    parser.add_argument("--date-from", help="Date interval begin (ex: 2019-12-23)")
    parser.add_argument("--date-to", help="Date interval End (ex: 2019-12-23)")
    return parser.parse_args()

args = parse_args()

lightmeter_api = f"{args.host}/api/"

def req_json(method):
    import requests

    r = requests.get(f"{lightmeter_api}/{method}?from={args.date_from}&to={args.date_to}")

    if r.status_code != 200 or r.headers['Content-Type'] != "application/json":
        raise Exception("Error Querying Lightmeter")

    return r.json()

def build_delivery_status():
    s = req_json("deliveryStatus")

    def f(status):
        for p in s:
            if p['Key'] == status:
                return p['Value']
        return 0

    return f

def list_domains_with_counts(title, method):
    l = req_json(method)

    print(f"\n{title}:\n")
    for e in l:
        print(f"{e['Key']} {e['Value']}")
    
del_status = build_delivery_status()

print("Summary:\n")

print(f"{del_status('sent')} Sent")
print(f"{del_status('bounced')} Bounced")
print(f"{del_status('deferred')} Deferred")

list_domains_with_counts("Busiest Domains", "topBusiestDomains")
list_domains_with_counts("Top Deferred Domains", "topDeferredDomains")
list_domains_with_counts("Top Bounced Domains", "topBouncedDomains")
