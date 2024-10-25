#! /usr/bin/env python3

# Create multiple PCPU loads, each random min..max %, spread over random
# min..max #threads and running for a min..max duration.

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
DEFAULT_MIN_THREADS = 1
DEFAULT_MAX_THREADS = 0  # i.e. constant #thread
DEFAULT_MIN_PCPU = 1
DEFAULT_MAX_PCPU = 0  # i.e. constant %CPU
DEFAULT_MIN_RUNTIME_SEC = 15
DEFAULT_MAX_RUNTIME_SEC = 30

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
        "--min-threads",
        default=DEFAULT_MIN_THREADS,
        type=int,
        metavar="MIN_T",
        help="""The minimum number of threads per process. Default: %(default)s""",
    )
    parser.add_argument(
        "-T",
        "--max-threads",
        default=DEFAULT_MAX_THREADS,
        type=int,
        metavar="MAX_T",
        help="""The maximum number of threads per process. If <= 0 then
             MIN_T is used as constant #threads. Default: %(default)s""",
    )
    parser.add_argument(
        "-p",
        "--min-pcpu",
        default=DEFAULT_MIN_PCPU,
        metavar="MIN_PCPU",
        type=float,
        help="""The minimum %CPU, per process. The usage will be spread evenly
             across the threads. Default: %(default)s""",
    )
    parser.add_argument(
        "-P",
        "--max-pcpu",
        default=DEFAULT_MAX_PCPU,
        metavar="MAX_PCPU",
        type=float,
        help="""The maximum %CPU, per process. If <= 0 then MIN_PCPU is used
             as constant load. Default: %(default)s""",
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
        help="""Max runtime for the process, in sec. If <= 0 then MIN_SEC is
             used as runtime in all cases. Default: %(default)s""",
    )

    args = parser.parse_args()
    num_proc = args.num_proc
    min_threads, max_threads = args.min_threads, args.max_threads
    min_pcpu, max_pcpu = args.min_pcpu, args.max_pcpu
    min_runtime, max_runtime = args.min_runtime, args.max_runtime

    for signum in [signal.SIGINT, signal.SIGHUP, signal.SIGTERM]:
        signal.signal(signum, sig_handler)

    running = {}
    while True:
        while len(running) < num_proc:
            if max_threads > min_threads:
                num_threads = int(
                    min_threads + random.random() * (max_threads - min_threads + 1)
                )
                num_threads = min(num_threads, max_threads)
            else:
                num_threads = min_threads
            if max_pcpu > min_pcpu:
                target_pcpu = min_pcpu + random.random() * (max_pcpu - min_pcpu)
            else:
                target_pcpu = min_pcpu
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
                f"--target-pcpu={target_pcpu:.1f}",
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
