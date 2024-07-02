import subprocess
import json


def run_lshw(cls=""):
    cmd = "lshw -quiet -json"
    if cls != "":
        cmd += f" -C {cls}"
    out = subprocess.run(cmd, shell=True, text=True,
                         stdout=subprocess.PIPE).stdout
    return json.loads(out)


def fetch_gpu_info():
    items = run_lshw("display")
    res = []
    for item in items:
        d = {
            "address": item["businfo"].split("@")[1],
            "product": item["product"],
            "vendor": item["vendor"],
        }
        res.append(d)
    return res


def fetch_cpuinfo():
    items = run_lshw("processor")
    res = []
    for item in items:
        d = {
            "address": item["businfo"].split("@")[1],
        }
        res.append(d)
        return res


def add_gpu_resource_to_eru(hostname):
    gpus = fetch_gpu_info()
    cmd_extra_args = {
        "gpu": {
            "gpu_map": {g["address"]: g for g in gpus},
        }
    }
    cmd = f"/usr/local/bin/eru-cli node set --cpu 72 --memory 62G --storage 62G --extra-resources '{json.dumps(cmd_extra_args)}'"
    print(cmd)

    out = subprocess.run(cmd, shell=True, text=True,
                         stdout=subprocess.PIPE).stdout
    print(out)


if __name__ == "__main__":
    # run_lshw()
    # print(run_lshw())
    # print(fetch_gpu_info())
    add_gpu_resource_to_eru("192-168-68-14")
