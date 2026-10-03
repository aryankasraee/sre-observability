"""Steady synthetic traffic against shop-api: ~20 requests/second."""
import itertools, os, time, urllib.error, urllib.request

BASE = os.environ.get("TARGET", "http://shop-api:8080")
ROUTES = ["/checkout", "/catalog", "/catalog", "/catalog"]
INTERVAL = float(os.environ.get("INTERVAL", "0.05"))

for route in itertools.cycle(ROUTES):
    try:
        urllib.request.urlopen(BASE + route, timeout=3).read()
    except urllib.error.HTTPError:
        pass  # 5xx is expected while chaos is on; the app counts it
    except Exception as e:  # app restarting, etc.
        print("request failed:", e, flush=True)
    time.sleep(INTERVAL)
