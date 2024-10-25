#! /usr/bin/env python3

# Create a %CPU load, 0..100%, spread over T threads, for a given duration.

import argparse
import os
import threading
import time
from typing import Optional

DEFAULT_NUM_THREADS = 1
DEFAULT_TARGET_PCPU = 1
DEFAULT_RUNTIME = None
CYCLE_INTERVAL_SEC = 0.1
CORRECTION_INTERVAL_SEC = 1


class CorrectionFactor:
    def __init__(
        self,
        target_pcpu: float = DEFAULT_TARGET_PCPU,
        interval_sec: float = CORRECTION_INTERVAL_SEC,
        alpha: float = 0.5,  # for exponential decay
    ):
        self._target_pcpu = target_pcpu
        self._interval_sec = interval_sec
        self._lck = threading.RLock()
        self._factor = 1
        self._alpha = alpha if 0 < alpha and alpha < 1 else 1

    def get(self):
        with self._lck:
            return self._factor

    def loop(self, runtime_sec: Optional[float] = None):
        prev_times, prev_ts = os.times(), time.time()
        next_ts = time.time() + self._interval_sec
        prev_f, curr_f = 1 - self._alpha, self._alpha
        deadline = time.time() + runtime_sec if runtime_sec is not None else None
        while deadline is None or time.time() <= deadline:
            if next_ts > deadline:
                next_ts = deadline
            pause = next_ts - time.time()
            if pause > 0:
                time.sleep(pause)
            next_ts += self._interval_sec
            curr_times, curr_ts = os.times(), time.time()
            curr_pcpu = (
                (curr_times[0] - prev_times[0] + curr_times[1] - prev_times[1])
                / (curr_ts - prev_ts)
                * 100
            )
            curr_factor = self._target_pcpu / curr_pcpu if curr_pcpu > 0 else 1
            with self._lck:
                self._factor = curr_f * curr_factor + prev_f * self._factor
            prev_times, prev_ts = curr_times, curr_ts


def make_cpu_load(
    target_pcpu: float = DEFAULT_TARGET_PCPU,
    cycle_interval_sec: float = CYCLE_INTERVAL_SEC,
    runtime_sec: Optional[float] = None,
    cf: Optional[CorrectionFactor] = None,
):
    if target_pcpu <= 0:
        # Simply block:
        if runtime_sec is not None and runtime_sec > 0:
            # For the duration
            time.sleep(runtime_sec)
        else:
            # Forever:
            with threading.Condition() as c:
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
        if cf is not None:
            crunch_deadline = time.time() + crunch_interval * cf.get()
        else:
            crunch_deadline = time.time() + crunch_interval


def run_proc(
    num_threads: int = DEFAULT_NUM_THREADS,
    target_pcpu: float = DEFAULT_TARGET_PCPU,
    cycle_interval_sec: float = CYCLE_INTERVAL_SEC,
    runtime_sec: Optional[float] = None,
):
    if num_threads == 1:
        return make_cpu_load(
            target_pcpu=target_pcpu,
            cycle_interval_sec=cycle_interval_sec,
            runtime_sec=runtime_sec,
        )

    threads = []
    cf = CorrectionFactor(target_pcpu=target_pcpu)
    make_cpu_load_args = dict(
        target_pcpu=target_pcpu / num_threads,
        cycle_interval_sec=cycle_interval_sec,
        runtime_sec=runtime_sec,
        cf=cf,
    )
    for _ in range(num_threads):
        t = threading.Thread(
            target=make_cpu_load,
            kwargs=make_cpu_load_args,
            daemon=True,
        )
        t.start()
        threads.append(t)
    cf.loop(runtime_sec=runtime_sec)


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
