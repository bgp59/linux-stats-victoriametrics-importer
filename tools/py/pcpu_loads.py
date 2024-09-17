#! /usr/bin/env python3

# Create multiple %CPU load, each 0..100%, spread over T threads and running for a (random) duration.

import argparse
import os
import random
import signal
import subprocess
import sys
import time
from typing import Optional

from pcpu_load import CYCLE_INTERVAL_SEC

DEFAULT_NUM_PROC = 1
DEFAULT_NUM_THREADS = 1
DEFAULT_TARGET_PCPU = 1
DEFAULT_MIN_RUNTIME_SEC = 5
DEFAULT_MAX_RUNTIME_SEC = 15

LOAD_UTIL = "pcpu_load.py"


this_dir = os.path.dirname(os.path.abspath(__file__))
os.environ["PATH"] = ".:" + os.environ.get("PATH", "")


def ts(t: Optional[float] = None) -> str:
    if t is None:
        t = time.time()
    millisec = int((t - int(t)) * 1000)
    return time.strftime(f"%Y-%m-%dT%H:%M:%S.{millisec:03d}%z", time.localtime(t))


def sig_handler(signum, frame):
    print(f"[{ts()}] Received {signal.Signals(signum)._name_}", file=sys.stderr)
    cleanup_cmd = ["pkill", "-KILL", "-P", str(os.getpid())]
    print(f"[{ts()}] Executing cleanup '{' '.join(cleanup_cmd)}'", file=sys.stderr)
    subprocess.run(cleanup_cmd)
    sys.exit(-signum)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-n",
        "--num-proc",
        default=DEFAULT_NUM_PROC,
        type=int,
        metavar="N",
        help="""The number of processes. Default: %(default)s""",
    )
    parser.add_argument(
        "-t",
        "--num-threads",
        default=DEFAULT_NUM_THREADS,
        type=int,
        metavar="T",
        help="""The number of threads per process. Default: %(default)s""",
    )
    parser.add_argument(
        "-p",
        "--target-pcpu",
        default=DEFAULT_TARGET_PCPU,
        metavar="%CPU",
        type=float,
        help="""The target %%CPU, per process. The usage will be spread evenly
             across the threads. Default: %(default)s""",
    )
    parser.add_argument(
        "-r",
        "--min-runtime",
        default=DEFAULT_MIN_RUNTIME_SEC,
        metavar="MIN_SEC",
        type=float,
        help="""Min runtime for the process, in sec. Default: %(default)s""",
    )
    parser.add_argument(
        "-R",
        "--max-runtime",
        default=DEFAULT_MAX_RUNTIME_SEC,
        metavar="MAX_SEC",
        type=float,
        help="""Max runtime for the process, in sec. If <= 0 then min-runtime is
             used as runtime in all cases. Default: %(default)s""",
    )

    args = parser.parse_args()
    num_proc = args.num_proc
    num_threads = args.num_threads
    target_pcpu = args.target_pcpu
    min_runtime = args.min_runtime
    max_runtime = args.max_runtime

    for signum in [signal.SIGINT, signal.SIGHUP, signal.SIGTERM]:
        signal.signal(signum, sig_handler)

    running = {}
    while True:
        while len(running) < num_proc:
            if max_runtime > min_runtime:
                runtime = (
                    int(
                        (min_runtime + random.random() * (max_runtime - min_runtime))
                        / CYCLE_INTERVAL_SEC
                    )
                    * CYCLE_INTERVAL_SEC
                )
            else:
                runtime = min_runtime
            args = [
                LOAD_UTIL,
                f"--num-threads={num_threads}",
                f"--target-pcpu={target_pcpu}",
                f"--runtime={runtime:.2f}",
            ]
            p = subprocess.Popen(
                args, cwd=this_dir, stdin=subprocess.DEVNULL, start_new_session=True
            )
            running[p.pid] = p
            print(f"[{ts()}] PID: {p.pid}, cmd: {' '.join(args)}", file=sys.stderr)
        pid, exit_sig = os.wait()
        exit_code, sig = exit_sig >> 8, exit_sig & 0xFF
        print(
            f"[{ts()}] PID: {p.pid}, exit code: {exit_code}, signal: {sig}",
            file=sys.stderr,
        )
        del running[pid]
