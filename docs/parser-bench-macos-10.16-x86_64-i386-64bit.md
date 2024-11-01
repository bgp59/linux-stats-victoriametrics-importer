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
| DiskstatsParser | BenchmarkDiskstatsParserIO<br>BenchmarkDiskstatsParser<br>BenchmarkDiskstatsParserProm | 71308<br>61944<br>10000 | 16573<br>20209<br>103287 | 152<br>336<br>14744 | 3<br>38<br>176 |
| InterruptsParser | BenchmarkInterruptsParserIO<br>BenchmarkInterruptsParser<br>BenchmarkInterruptsParserProm | 70729<br>59065<br>26955 | 16554<br>20506<br>45559 | 152<br>240<br>26093 | 3<br>35<br>170 |
| MountinfoParser | BenchmarkMountinfoParserIO<br>BenchmarkMountinfoParser/forceUpdate=false<br>BenchmarkMountinfoParser/forceUpdate=true | 70057<br>74232<br>45886 | 16496<br>16565<br>26917 | 152<br>176<br>10256 | 3<br>4<br>39 |
| NetDevParser | BenchmarkNetDevParserIO<br>BenchmarkNetDevParser<br>BenchmarkNetDevParserProm | 64890<br>64981<br>54165 | 17661<br>18701<br>22632 | 136<br>168<br>5896 | 3<br>6<br>16 |
| NetSnmp6Parser | BenchmarkNetSnmp6ParserIO<br>BenchmarkNetSnmp6Parser<br>BenchmarkNetSnmp6ParserProm | 63986<br>56462<br>22430 | 18087<br>21609<br>51283 | 152<br>176<br>20040 | 3<br>4<br>275 |
| NetSnmpParser | BenchmarkNetSnmpParserIO<br>BenchmarkNetSnmpParser<br>BenchmarkNetSnmpParserProm | 65606<br>60501<br>33686 | 17859<br>20293<br>35802 | 136<br>160<br>11960 | 3<br>4<br>117 |
| PidCmdlineParser | BenchmarkPidCmdlineParserIO<br>BenchmarkPidCmdlineParser | 71528<br>71979 | 16436<br>16979 | 152<br>272 | 3<br>6 |
| PidStatAllParser | BenchmarkPidStatAllParserIO/NFiles=241<br>BenchmarkPidStatAllParser/NPidTid=241<br>BenchmarkPidStatAllParserProm/NPidTid=241 | 285<br>274<br>153 | 4120040<br>4733535<br>7424392 | 35229<br>55133<br>381359 | 723<br>1205<br>7232 |
| PidStatParser | BenchmarkPidStatParserIO<br>BenchmarkPidStatParser<br>BenchmarkPidStatParserProm | 69811<br>66925<br>44216 | 16566<br>17430<br>26658 | 152<br>248<br>1336 | 3<br>5<br>31 |
| PidStatusAllParser | BenchmarkPidStatusAllParserIO/NFiles=241<br>BenchmarkPidStatusAllParser/NPidTid=241<br>BenchmarkPidStatusAllParserProm/NPidTid=241 | 285<br>238<br>138 | 4228012<br>4939057<br>8640068 | 36532<br>64593<br>2197618 | 723<br>1620<br>24787 |
| PidStatusParser | BenchmarkPidStatusParserIO<br>BenchmarkPidStatusParser<br>BenchmarkPidStatusParserProm | 69189<br>61730<br>37590 | 16888<br>20390<br>32367 | 152<br>272<br>9224 | 3<br>6<br>102 |
| SoftirqsParser | BenchmarkSoftirqsParserIO<br>BenchmarkSoftirqsParser<br>BenchmarkSoftirqsParserProm | 72056<br>63957<br>38480 | 16140<br>19112<br>30923 | 136<br>200<br>14992 | 3<br>13<br>42 |
| StatParser | BenchmarkStatParserIO<br>BenchmarkStatParser<br>BenchmarkStatParserProm | 71556<br>36801<br>19180 | 16120<br>32855<br>63068 | 136<br>160<br>47666 | 3<br>4<br>78 |

Notes:

  1. `IO` suffix designates the benchmark for reading the file into a buffer
  2. `Prom` suffix designates the benchmark for the official Prometheus [procfs](https://github.com/prometheus/procfs) parsers
  3. No suffix designates the benchmark for the custom parsers

