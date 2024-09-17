#! /usr/bin/env python3

# Create a %CPU load, 0..100%, spread over T threads and for a given duration.

import argparse
import threading
import time
from typing import Optional

DEFAULT_NUM_THREADS = 1
DEFAULT_TARGET_PCPU = 1
DEFAULT_RUNTIME = None
CYCLE_INTERVAL_SEC = 0.1


def make_cpu_load(
    target_pcpu: float = DEFAULT_TARGET_PCPU,
    cycle_interval_sec: float = CYCLE_INTERVAL_SEC,
    runtime_sec: Optional[float] = None,
):
    if target_pcpu <= 0:
        # Simply block:
        if runtime_sec is not None and runtime_sec > 0:
            # For the duration
            time.sleep(runtime_sec)
        else:
            # Forever:
            c = threading.Condition()
            c.acquire()
            c.wait()
        return

    crunch_interval = target_pcpu / 100 * cycle_interval_sec
    runtime_deadline = None if runtime_sec is None else time.time() + runtime_sec

    next_check = time.time() + cycle_interval_sec
    crunch_deadline = time.time() + crunch_interval
    while runtime_deadline is None or time.time() <= runtime_deadline:
        if time.time() <= crunch_deadline:
            continue
        pause = next_check - time.time()
        if pause > 0:
            time.sleep(pause)
        next_check += cycle_interval_sec
        crunch_deadline = time.time() + crunch_interval


def run_proc(
    num_threads: int = DEFAULT_NUM_THREADS,
    target_pcpu: float = DEFAULT_TARGET_PCPU,
    cycle_interval_sec: float = CYCLE_INTERVAL_SEC,
    runtime_sec: Optional[float] = None,
):
    if num_threads > 1:
        thread_target_cpu = target_pcpu / num_threads
    else:
        thread_target_cpu = target_pcpu
    threads = []
    for _ in range(num_threads - 1):
        t = threading.Thread(
            target=make_cpu_load,
            kwargs=dict(
                target_pcpu=thread_target_cpu,
                cycle_interval_sec=cycle_interval_sec,
                runtime_sec=runtime_sec,
            ),
            daemon=True,
        )
        t.start()
        threads.append(t)
    make_cpu_load(
        target_pcpu=thread_target_cpu,
        cycle_interval_sec=cycle_interval_sec,
        runtime_sec=runtime_sec,
    )


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-t",
        "--num-threads",
        default=DEFAULT_NUM_THREADS,
        type=int,
        metavar="T",
        help="""The number of threads per process. Default: %(default)s""",
    )
    parser.add_argument(
        "-P",
        "--target-pcpu",
        default=DEFAULT_TARGET_PCPU,
        metavar="%CPU",
        type=float,
        help="""The target %%CPU, per process. The usage will be spread evenly
             across the threads. Default: %(default)s""",
    )
    parser.add_argument(
        "-r",
        "--runtime",
        default=DEFAULT_RUNTIME,
        type=float,
        help="""Run time for the process, in sec, if not specified then run forever. Default: %(default)s""",
    )

    args = parser.parse_args()
    run_proc(
        num_threads=args.num_threads,
        target_pcpu=args.target_pcpu,
        runtime_sec=args.runtime,
    )
