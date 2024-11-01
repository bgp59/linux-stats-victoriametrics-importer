# Parser Benchmark Results

## Conditions

| Cond | Value |
| --- | --- |
| CPU | Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz |
| GOARCH | amd64 |
| GOOS | darwin |

## Results

| Parser | Benchmark | Run# | ns/op | Bytes/op | Allocs/op |
| :--- | :--- | ---: | ---: | ---: | ---: |
| DiskstatsParser | BenchmarkDiskstatsParserIO<br>BenchmarkDiskstatsParser<br>BenchmarkDiskstatsParserProm | 70154<br>61028<br>10000 | 16461<br>20132<br>102939 | 152<br>336<br>14744 | 3<br>38<br>176 |
| InterruptsParser | BenchmarkInterruptsParserIO<br>BenchmarkInterruptsParser<br>BenchmarkInterruptsParserProm | 71835<br>58827<br>25879 | 16307<br>20950<br>45322 | 152<br>240<br>26093 | 3<br>35<br>170 |
| MountinfoParser | BenchmarkMountinfoParser/forceUpdate=false<br>BenchmarkMountinfoParserIO<br>BenchmarkMountinfoParser/forceUpdate=true | 72984<br>72717<br>44824 | 16706<br>16709<br>26427 | 176<br>152<br>10256 | 4<br>3<br>39 |
| NetDevParser | BenchmarkNetDevParserIO<br>BenchmarkNetDevParser<br>BenchmarkNetDevParserProm | 65758<br>63169<br>50896 | 17766<br>18584<br>22860 | 136<br>168<br>5896 | 3<br>6<br>16 |
| NetSnmp6Parser | BenchmarkNetSnmp6ParserIO<br>BenchmarkNetSnmp6Parser<br>BenchmarkNetSnmp6ParserProm | 64221<br>57373<br>22874 | 18082<br>21643<br>51173 | 152<br>176<br>20040 | 3<br>4<br>275 |
| NetSnmpParser | BenchmarkNetSnmpParserIO<br>BenchmarkNetSnmpParser<br>BenchmarkNetSnmpParserProm | 63522<br>61168<br>33651 | 18203<br>20058<br>35067 | 136<br>160<br>11960 | 3<br>4<br>117 |
| PidCmdlineParser | BenchmarkPidCmdlineParserIO<br>BenchmarkPidCmdlineParser | 70894<br>73012 | 16535<br>17015 | 152<br>272 | 3<br>6 |
| PidStatAllParser | BenchmarkPidStatAllParserIO/NFiles=241<br>BenchmarkPidStatAllParser/NPidTid=241<br>BenchmarkPidStatAllParserProm/NPidTid=241 | 280<br>272<br>158 | 4101111<br>4328064<br>7547307 | 35229<br>55133<br>381360 | 723<br>1205<br>7232 |
| PidStatParser | BenchmarkPidStatParserIO<br>BenchmarkPidStatParser<br>BenchmarkPidStatParserProm | 69868<br>67389<br>44677 | 16698<br>17365<br>26941 | 152<br>248<br>1336 | 3<br>5<br>31 |
| PidStatusAllParser | BenchmarkPidStatusAllParserIO/NFiles=241<br>BenchmarkPidStatusAllParser/NPidTid=241<br>BenchmarkPidStatusAllParserProm/NPidTid=241 | 276<br>243<br>139 | 4199891<br>4991283<br>8577340 | 36532<br>64593<br>2197618 | 723<br>1620<br>24787 |
| PidStatusParser | BenchmarkPidStatusParserIO<br>BenchmarkPidStatusParser<br>BenchmarkPidStatusParserProm | 69567<br>60847<br>37309 | 16890<br>19984<br>31832 | 152<br>272<br>9224 | 3<br>6<br>102 |
| SoftirqsParser | BenchmarkSoftirqsParserIO<br>BenchmarkSoftirqsParser<br>BenchmarkSoftirqsParserProm | 71964<br>62731<br>35374 | 16409<br>18799<br>32551 | 136<br>200<br>14992 | 3<br>13<br>42 |
| StatParser | BenchmarkStatParserIO<br>BenchmarkStatParser<br>BenchmarkStatParserProm | 70902<br>35968<br>20108 | 16241<br>32687<br>61286 | 136<br>160<br>47666 | 3<br>4<br>78 |

Notes:

  1. `IO` suffix designates the benchmark for reading the file into a buffer
  2. `Prom` suffix designates the benchmark for the official [prometheus/procfs](https://github.com/prometheus/procfs) parsers
  3. No suffix designates the benchmark for the custom parsers

