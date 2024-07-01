import sys
import subprocess
import json
import time
import base64
from collections import namedtuple


def parse_args():
    Args = namedtuple('Args', ['domain', 'cmds'])

    for i in range(len(sys.argv)):
        if sys.argv[i] == "--domain":
            domain = sys.argv[i+1]
            cmds = sys.argv[i+2:]
            return Args(domain, cmds)


def run_qmp_cmd(domain, qmp_cmd):
    cmd = ["virsh", "qemu-agent-command",
           "--domain", domain, json.dumps(qmp_cmd)]
    out = subprocess.run(cmd, capture_output=True, text=True).stdout

    return json.loads(out)


if __name__ == "__main__":
    args = parse_args()
    if args is None or len(args.cmds) == 0:
        print("invalid domain or args")
        sys.exit(-1)

    qmp_cmd = {
        "execute": "guest-exec",
        "arguments": {
            "path": args.cmds[0],
            "arg": args.cmds[1:],
            "capture-output": True
        }
    }
    res = run_qmp_cmd(args.domain, qmp_cmd)
    pid = res["return"]["pid"]

    qmp_cmd = {
        "execute": "guest-exec-status",
        "arguments": {"pid": pid}
    }
    n = 180
    exited = False
    while (not exited) and n > 0:
        time.sleep(2)
        res = run_qmp_cmd(args.domain, qmp_cmd)
        exited = res["return"]["exited"]
        n -= 1

    exitcode = res["return"]["exitcode"]
    if exitcode != 0:
        err_data = res["return"]["err-data"]
        print(base64.b64decode(err_data).decode('utf-8'))
        sys.exit(-1)
    out_data = res["return"]["out-data"]
    print(base64.b64decode(out_data).decode('utf-8'))
