# Parser Benchmark Results

## Conditions

| Cond | Value |
| --- | --- |
| CPU | Apple M3 |
| GOARCH | arm64 |
| GOOS | darwin |

## Results

| Parser | Benchmark | Run# | ns/op | Bytes/op | Allocs/op |
| :--- | :--- | ---: | ---: | ---: | ---: |
| DiskstatsParser | BenchmarkDiskstatsParserIO<br>BenchmarkDiskstatsParser<br>BenchmarkDiskstatsParserProm | 93397<br>72454<br>12900 | 12854<br>16382<br>93568 | 152<br>336<br>15128 | 3<br>38<br>176 |
| InterruptsParser | BenchmarkInterruptsParserIO<br>BenchmarkInterruptsParser<br>BenchmarkInterruptsParserProm | 87865<br>72271<br>33009 | 12837<br>16987<br>36460 | 152<br>240<br>26324 | 3<br>35<br>171 |
| MountinfoParser | BenchmarkMountinfoParserIO<br>BenchmarkMountinfoParser/forceUpdate=false<br>BenchmarkMountinfoParser/forceUpdate=true | 94684<br>96782<br>57996 | 12968<br>13009<br>21103 | 152<br>176<br>10256 | 3<br>4<br>39 |
| NetDevParser | BenchmarkNetDevParserIO<br>BenchmarkNetDevParser<br>BenchmarkNetDevParserProm | 65162<br>67764<br>59604 | 17988<br>18576<br>20688 | 136<br>168<br>5896 | 3<br>6<br>16 |
| NetSnmp6Parser | BenchmarkNetSnmp6ParserIO<br>BenchmarkNetSnmp6Parser<br>BenchmarkNetSnmp6ParserProm | 67117<br>55129<br>27294 | 18092<br>21413<br>43966 | 152<br>176<br>20040 | 3<br>4<br>275 |
| NetSnmpParser | BenchmarkNetSnmpParserIO<br>BenchmarkNetSnmpParser<br>BenchmarkNetSnmpParserProm | 62631<br>59682<br>36613 | 17839<br>20324<br>32523 | 136<br>160<br>11960 | 3<br>4<br>117 |
| PidCmdlineParser | BenchmarkPidCmdlineParserIO<br>BenchmarkPidCmdlineParser | 89788<br>92835 | 12820<br>13214 | 152<br>272 | 3<br>6 |
| PidStatAllParser | BenchmarkPidStatAllParserIO/NFiles=241<br>BenchmarkPidStatAllParser/NPidTid=241<br>BenchmarkPidStatAllParserProm/NPidTid=241 | 368<br>357<br>192 | 3175892<br>3344697<br>6172055 | 35228<br>55132<br>381359 | 723<br>1205<br>7232 |
| PidStatParser | BenchmarkPidStatParserIO<br>BenchmarkPidStatParser<br>BenchmarkPidStatParserProm | 92742<br>82893<br>54816 | 12933<br>13629<br>22032 | 152<br>248<br>1336 | 3<br>5<br>31 |
| PidStatusAllParser | BenchmarkPidStatusAllParserIO/NFiles=241<br>BenchmarkPidStatusAllParser/NPidTid=241<br>BenchmarkPidStatusAllParserProm/NPidTid=241 | 354<br>298<br>175 | 3264585<br>3880848<br>6853190 | 36530<br>64593<br>2136402 | 723<br>1620<br>24538 |
| PidStatusParser | BenchmarkPidStatusParserIO<br>BenchmarkPidStatusParser<br>BenchmarkPidStatusParserProm | 90650<br>72498<br>48866 | 13066<br>15995<br>23912 | 152<br>272<br>8936 | 3<br>6<br>101 |
| SoftirqsParser | BenchmarkSoftirqsParserIO<br>BenchmarkSoftirqsParser<br>BenchmarkSoftirqsParserProm | 95894<br>77880<br>50244 | 12647<br>15572<br>22990 | 136<br>200<br>14992 | 3<br>13<br>42 |
| StatParser | BenchmarkStatParserIO<br>BenchmarkStatParser<br>BenchmarkStatParserProm | 96032<br>39738<br>24812 | 12666<br>30936<br>48767 | 136<br>160<br>47666 | 3<br>4<br>78 |

Notes:

  1. `IO` suffix designates the benchmark for reading the file into a buffer
  2. `Prom` suffix designates the benchmark for the official [prometheus/procfs](https://github.com/prometheus/procfs) parsers
  3. No suffix designates the benchmark for the custom parsers

