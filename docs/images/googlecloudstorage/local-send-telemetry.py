import json,time,urllib.request
resource={'attributes':[{'key':'service.name','value':{'stringValue':'core36-gcs-e2e'}},{'key':'validation.ticket','value':{'stringValue':'CORE-36'}}]}
start=time.time_ns()
spans=[{'traceId':format(i+1,'032x'),'spanId':format(i+1,'016x'),'name':'gcs-archive-validation-'+str(i),'kind':1,'startTimeUnixNano':str(start),'endTimeUnixNano':str(start+1000000),'attributes':[{'key':'validation.record','value':{'intValue':str(i)}}]} for i in range(12)]
logs=[{'timeUnixNano':str(start+i),'severityNumber':9,'severityText':'INFO','body':{'stringValue':'CORE-36 GCS archive validation log '+str(i)},'attributes':[{'key':'validation.record','value':{'intValue':str(i)}}]} for i in range(12)]
payloads={'traces':{'resourceSpans':[{'resource':resource,'scopeSpans':[{'scope':{'name':'core36-validation'},'spans':spans}]}]},'logs':{'resourceLogs':[{'resource':resource,'scopeLogs':[{'scope':{'name':'core36-validation'},'logRecords':logs}]}]}}
for signal,payload in payloads.items():
 request=urllib.request.Request('http://127.0.0.1:15438/v1/'+signal,data=json.dumps(payload).encode(),headers={'Content-Type':'application/json'},method='POST')
 with urllib.request.urlopen(request) as response: print(signal,response.status,response.read().decode())
