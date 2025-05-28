
راه اندازی GatewayAPI
داستان از این قرار که ما برای Network‌ از Cilium‌ استفاده کردیم. برای نصب بخش های مختلف یک سری اقدامات و Config ها و روش هایی رو در پیش گرفتیم که من در ادامه به اون ها اشاره میکنم:
توضیحات
فلسفه این نمودار…‍!


نصب Cilium
https://docs.cilium.io/en/stable/installation/k8s-install-helm/

نصب GatewayAPI
$ kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.1.0/config/crd/standard/gateway.networking.k8s.io_gatewayclasses.yaml
$ kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.1.0/config/crd/standard/gateway.networking.k8s.io_gateways.yaml
$ kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.1.0/config/crd/standard/gateway.networking.k8s.io_httproutes.yaml
$ kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.1.0/config/crd/standard/gateway.networking.k8s.io_referencegrants.yaml
$ kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.1.0/config/crd/standard/gateway.networking.k8s.io_grpcroutes.yaml
$ kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.1.0/config/crd/experimental/gateway.networking.k8s.io_tlsroutes.yaml



فعال سازی GatewayAPI در Cilium
helm upgrade cilium cilium/cilium --version 1.17.1 \
    --namespace kube-system \
    --reuse-values \
    --set kubeProxyReplacement=true \
    --set gatewayAPI.enabled=true \
    --set operator.replicas=1 \
    --set hostRootfsMount.enabled=true \
    --set securityContext.privileged=true
$ kubectl -n kube-system rollout restart deployment/cilium-operator
$ kubectl -n kube-system rollout restart ds/cilium

نکته: اگر cilium رو روی یک node قصد داشتیم بیاریم بالا ( مثلا روی کلاستر تستی) مقداری که قرمز کردم رو موقع نصب باید وارد کنیم. چون سه تا operator میاره بالا در حالت تک node دو تا از operator ها خطای زیر رو میدن. برای همین کلا یکی میاریم بالا که تست هامون رو بگیریم.
Warning FailedScheduling 3m6s default-scheduler 0/1 nodes are available: 1 node(s) didn't have free ports for the requested pod ports. preemption: 0/1 nodes are available: 1 No preemption victims found for incoming pod.

فعال سازی nodeIPAM در Cilium
helm upgrade cilium cilium/cilium --version 1.16.3 \
  --namespace kube-system \
  --reuse-values \
  --set nodeIPAM.enabled=true
kubectl -n kube-system rollout restart deployment/cilium-operator




تعریف gateway.yml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: my-gateway
  annotations:
    "lbipam.cilium.io/ips": "192.168.100.60,192.168.100.62"
spec:
  gatewayClassName: cilium
  listeners:
  - protocol: HTTP
    port: 80
    name: web-gw
    allowedRoutes:
      namespaces:
        from: Same


تعریف ippool.yaml
apiVersion: cilium.io/v2alpha1
kind: CiliumLoadBalancerIPPool
metadata:
  name: default-pool
spec:
  blocks:
    #
- cidr: 192.168.100.60/32
    - start: "192.168.100.60"
      stop: "192.168.100.62"



تعریف یک پاد (manifest.yml)
├── whoami-route.yaml
└── whoami.yaml
whoami-route.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: whoami
  labels:
    app: whoami
spec:
  selector:
    matchLabels:
      app: whoami
  replicas: 1
  template:
    metadata:
      labels:
        app: whoami
    spec:
      containers:
      - name: whoami
        image: traefik/whoami
        ports:
        - containerPort: 80
        resources:
          limits:
            cpu: 10m
          requests:
            cpu: 5m
#        imagePullPolicy: Never
---
apiVersion: v1
kind: Service
metadata:
  name: whoami
  labels:
    app: whoami
    service: whoami
spec:
  ports:
  - port: 80
    name: http
  selector:
    app: whoami
---
apiVersion: autoscaling/v1
kind: HorizontalPodAutoscaler
metadata:
 name: hpa-whoami
spec:
 scaleTargetRef:
   apiVersion: apps/v1
   kind: Deployment
   name: whoami
 minReplicas: 1
 maxReplicas: 10
 targetCPUUtilizationPercentage: 80


whoami-route.yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: http-app-1
spec:
  parentRefs:
  - name: my-gateway
    namespace: default
  hostnames:
  - demo.home.lab
  rules:
  - matches:
    #- path:
    #    type: PathPrefix
    #    value: /
    backendRefs:
    - name: whoami
      port: 80




نکته: init‌ اولیه کلاستر کوبرنتز باید مورد زیر رو به دلیل نصب cilium‌ رعایت کنیم.
تکمیل شود
جعبه ابزار

kubectl api-resources | grep CiliumLoadBalancerIPPool


kubectl get gatewayclass


kubectl describe gatewayclass cilium


 kubectl get services


 kubectl apply -f ippool.yaml


kubectl port-forward service/whoami 8585:80 --address=192.168.100.48


 kubectl get httproute -A


kubectl get gateway -A


 منابع
https://gateway-api.sigs.k8s.io/
https://kubernetes.io/docs/concepts/services-networking/gateway/
https://gateway-api.sigs.k8s.io/concepts/use-cases/
https://docs.cilium.io/en/stable/network/servicemesh/gateway-api/http/
https://docs.cilium.io/en/stable/network/node-ipam/
https://docs.cilium.io/en/stable/network/lb-ipam/#services

راه اندازی Kubernetes Storage
پیش نیاز
اضافه کردن Storage جدید
parted /dev/sdb mklabel gpt
fdisk /dev/sdb #make a partition 
pvcreate /dev/sdb1
vgcreate vg_data /dev/sdb1
lvcreate -l 100%FREE -n lv_storage vg_data
mkfs.ext4 /dev/mapper/vg_data-lv_storage
mount /dev/mapper/vg_data-lv_storage /data/
vim /etc/fstab
/dev/mapper/vg_data-lv_storage   /data  ext4   rw   0   0


نصب و مدیریت دسترسی های NFS
sudo apt install nfs-kernel-server
mkdir /data
chown nobody:nogroup /data
sudo vim /etc/exports
/data           192.168.100.0/24(rw,insecure,no_root_squash,sync,no_subtree_check)


روش نصب Dynamic provisioning using StorageClass
You must install the NFS provisioner to provision PersistentVolume dynamically using StorageClasses. I use the nfs-subdir-external-provisioner to achieve this. The following commands install everything we need using the Helm package manager.

helm repo add nfs-subdir-external-provisioner https://kubernetes-sigs.github.io/nfs-subdir-external-provisioner

helm install nfs-subdir-external-provisioner nfs-subdir-external-provisioner/nfs-subdir-external-provisioner \
  --create-namespace \
  --namespace nfs-provisioner \
  --set nfs.server=nfs-storage.infra.done.tech \
  --set nfs.path=/data
  --set storageClass.reclaimPolicy=Retain

نکته: در قسمت nfs.server یک نام دامنه استفاده کردیم. این نام دامنه در DNS میکروتیک اضافه شده است. IP این نام دامنه به سرور NFS اشاره میکند.

برای ایجاد یک PersistentVolumeClaim از  manifest زیراستفاده میکنیم:
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: nfs-test
  labels:
    storage.k8s.io/name: nfs
    storage.k8s.io/part-of: kubernetes-complete-reference
    storage.k8s.io/created-by: ssbostan
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: nfs-client
  resources:
    requests:
      storage: 1Gi


ایجاد یک Statefulset و اتصال آن به PVC
apiVersion: apps/v1                                                               
kind: StatefulSet                                                                 
metadata:                                                                         
  name: nfs-test-statefulset                                                      
  labels:                                                                         
    app: nfs-test                                                                 
spec:                                                                             
  serviceName: "nfs-test-service"  # Headless service name                        
  replicas: 1  # Number of pods to deploy                                         
  selector:                                                                       
    matchLabels:                                                                  
      app: nfs-test                                                               
  template:                                                                       
    metadata:                                                                     
      labels:                                                                     
        app: nfs-test                                                             
    spec:                                                                         
      containers:                                                                 
        - name: test-container                                                    
          image: nginx                                                            
          ports:                                                                  
            - containerPort: 80                                                   
          volumeMounts:                                                           
            - name: nfs-storage                                                   
              mountPath: /usr/share/nginx/html  # Mount path inside the container 
                                                 
  volumeClaimTemplates:  # This creates a PVC for each pod in the StatefulSet     
    - metadata:                                                                   
        name: nfs-storage  # Must match the volumeMount name                      
      spec:                                                                       
        accessModes:                                                              
          - ReadWriteMany                                                         
        storageClassName: nfs-client  # Use the NFS StorageClass                  
        resources:                                                                
          requests:                                                               
            storage: 1Gi

نکته: وقتی از یک statefulset استفاده میکنیم دیگر نیاز به ایجاد یک PersistentVolumeClaim به صورت مجزا نداریم. در manifest که برای statefulset ایجاد کردیم مقادیر مرتبط با PersistentVolumeClaim اضافه شده است.

سناریوی حذف یک PVC
 در صورتی که ما تعدادی پاد از نوع statefulset داشته باشیم که به یک pvc متصل شده باشند در صورت حذف statefulset و pvc با روش زیر میتوان مجدد pvc آن statefulset را به pv متصل کرد.
همانطور که در عکس زیر نشان داده شده است در صورتی که ما یک statfulset رو پاک کنیم و بدون edit فایل pv مجدد pv  فعلی را ایجاد کنیم، یک pv‌ جدید درست کرده و pvc جدید را نیز به آن assign میکند. با روش زیر میتوان همان pvc به statefull متصل کرد.
نکته: در عکس زیر به status‌ ها دقت کنید.

kubectl edit pv pv_name



کافی است که مقدار clamRef از به صورت کامل پاک کنیم و مجدد manifest مرتبط با statefulset را اجرا کنیم.(قسمت قرمز رنگ)
بعد از edit فایل pv

نکته: در صورتی که statefulset بالا باشد نیمتوان pvc را پاک کرد.
نکته:  برای اینکه ما بتونیم به صورت پیش فرض از این storage class استفاده کنیم میتونیم این storage class به default قرار بدیم. برای این کار کافی است از دستور زیر استفاده کنیم:
kubectl patch storageclass nfs-client -p '{"metadata": {"annotations": {"storageclass.kubernetes.io/is-default-class": "true"}}}'

جعبه ابزار
kubectl  get pv
kubectl get pvc
kubectl delete persistentvolumeclaims pvc-name
done@gateway:~/k8s-manifests/nfs$ kubectl get storageclasses.storage.k8s.io

منابع
https://kubedemy.io/kubernetes-storage-part-1-nfs-complete-tutorial
https://kubernetes.io/docs/concepts/storage/persistent-volumes/
https://kubernetes.io/docs/concepts/storage/persistent-volumes/#reclaim-policy
https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims
https://kubernetes.io/docs/concepts/storage/storage-classes/
https://kubernetes.io/docs/concepts/storage/volume-attributes-classes/
https://www.arvancloud.ir/en/dev/ips 

راه اندازی cert manager
نصب cert manager
helm install cert-manager jetstack/cert-manager --version v1.16.2 \
    --namespace cert-manager \
    --create-namespace \
    --set installCRDs=true \
    --set "extraArgs={--feature-gates=ExperimentalGatewayAPISupport=true}" \
    --set "extraArgs={--enable-gateway-api}"

نصب cert manager webhook arvan
helm repo add hbx https://hbahadorzadeh.github.io/helm-chart/
helm install -n cert-manager hbx/cert-manager-webhook-arvan


cert_manager.yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: wildcard-done
spec:
  secretName: wildcard-done
  duration: 2160h0m0s # 90d
  renewBefore: 360h0m0s # 15d
  issuerRef:
    name: letsencrypt-arvan
    kind: ClusterIssuer
  commonName: "*.done.tech"
  dnsNames:
    - "*.done.tech"
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: wildcard-infra-done
spec:
  secretName: wildcard-infra-done
  duration: 2160h0m0s # 90d
  renewBefore: 360h0m0s # 15d
  issuerRef:
    name: letsencrypt-arvan
    kind: ClusterIssuer
  commonName: "*.infra.done.tech"
  dnsNames:
    - "*.infra.done.tech"
---

apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-arvan
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: infra@done.tech
    privateKeySecretRef:
      name: letsencrypt-account-key-arvan
    solvers:
    - dns01:
        webhook:
          groupName: hbahadorzadeh.github # name of the group you setted at the start of this course
          solverName: arvancloud
          config:
            ttl: 120
            authApiSecretRef:
              name: "arvan-credentials"
              key:  "apikey"
            baseUrl: "https://napi.arvancloud.ir"



 secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: arvan-credentials
  namespace: cert-manager
type: Opaque
stringData:
  apikey: "XXXXXXX"


gateway.yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: websecure-gateway
spec:
  gatewayClassName: cilium
  addresses:
    - value: "192.168.100.60"
  listeners:
    - protocol: HTTPS
      port: 443
      name: gw-websecure-infra-done
      hostname: "*.infra.done.tech"
      tls:
        mode: Terminate
        certificateRefs:
          - kind: Secret
            name: wildcard-infra-done
      allowedRoutes:
        namespaces:
          from: All
    - protocol: HTTPS
      port: 443
      name: gw-websecure-done
      hostname: "*.done.tech"
      tls:
        mode: Terminate
        certificateRefs:
          - kind: Secret
            name: wildcard-done
      allowedRoutes:
        namespaces:
          from: All



whoami.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: whoami
  namespace: test
  labels:
    app: whoami
spec:
  selector:
    matchLabels:
      app: whoami
  replicas: 1
  template:
    metadata:
      labels:
        app: whoami
    spec:
      containers:
      - name: whoami
        image: traefik/whoami
        ports:
        - containerPort: 80
        resources:
          limits:
            cpu: 10m
          requests:
            cpu: 5m
---
apiVersion: v1
kind: Service
metadata:
  name: whoami
  namespace: test
  labels:
    app: whoami
    service: whoami
spec:
  ports:
  - port: 80
    name: http
  selector:
    app: whoami
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: http-app-1
  namespace: test
spec:
  parentRefs:
  - name: websecure-gateway
    namespace: default
  hostnames:
  - "test.done.tech"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: whoami
      port: 80
---


جعبه ابزار
kubectl get issuers
kubectl get clusterissuers
kubectl get certificaterequests
kubectl describe certificaterequests certificaterequests_name
kubectl describe certificate certificate_name
kubectl describe clusterissuer  clusterissuer _name
kubectl get orders
kubectl get challenges
kubectl logs -l app=cert-manager -n cert-manager
kubectl get pod -o yaml
kubectl get gateway -o yaml

منابع
https://cert-manager.io/docs/
https://docs.cilium.io/en/stable/network/servicemesh/gateway-api/https/
https://itnext.io/cilium-gateway-api-cert-manager-and-lets-encrypt-updates-cc730818cb17
https://kubito.dev/posts/gateway-api-cert-manager/
https://github.com/kiandigital/cert-manager-webhook-arvan
https://artifacthub.io/packages/helm/hbahadorzadeh/cert-manager-webhook-arvan


میکروتیک 

Steps to Update CDN Whitelist via WinboxSteps to Update CDN Whitelist via Winbox
Login to Your Router:
Open Winbox.
Connect to your router using its IP address, username, and password.
Create a Script:
Navigate to System > Scripts.
Click the + button to add a new script.
Fill in the following:
Name: UpdateCDNWhitelist
Policy: Ensure all policies (e.g., read, write, policy, test, fetch) are checked.
Source: Paste the corrected script into the Source field:



:local url "https://www.arvancloud.ir/en/ips.txt"
:local addressListName "arvan-cdn-allowed-addresses"
:local fileName "arvan-cdn-allowed-addresses.txt"


/tool fetch url=$url mode=https dst-path=$fileName
:if ([:len [/file find name=$fileName]] = 0) do={
    :log error "Failed to fetch CDN IP list. File not found."
    :error "File $fileName not found"
}


:local newIPs [/file get $fileName contents]


:local newIPsWithNewline "$newIPs\n"


:local newIPArray [:toarray ""]
:while ([:len $newIPsWithNewline] > 0) do={
    :local newlineIndex [:find $newIPsWithNewline "\n"]
    :if ($newlineIndex = nil) do={
        :log warning "Unexpected missing newline processing."
        :error "Failed to parse IP list"
    }
    :local line [:pick $newIPsWithNewline 0 $newlineIndex]
    :set newIPsWithNewline [:pick $newIPsWithNewline ($newlineIndex + 1) [:len $newIPsWithNewline]]
    :if ([:len $line] > 0) do={
        :set newIPArray ($newIPArray, $line)
    }
}



:foreach entry in=[/ip firewall address-list find list=$addressListName] do={
    :local currentIP [/ip firewall address-list get $entry address]
    :local found 0
    :foreach ip in=$newIPArray do={
        :if ($ip = $currentIP) do={
            :set found 1
        }
    }
    :if ($found = 0) do={
        /ip firewall address-list remove $entry
        :log info "Removed outdated IP: $currentIP"
    }
}



:foreach ip in=$newIPArray do={
    :if ([:len [/ip firewall address-list find list=$addressListName address=$ip]] = 0) do={
        /ip firewall address-list add list=$addressListName address=$ip comment="Auto-updated from ArvanCloud"
        :log info "Added new IP: $ip"
    }
}

:log info "CDN whitelist updated successfully."




Click OK to save the script.
Run the Script:
In the System > Scripts window, select the script you created (UpdateCDNWhitelist).
Click Run Script to execute it manually.
Monitor the Log section (Log in Winbox left menu) to confirm the script runs successfully.
Schedule the Script:
Navigate to System > Scheduler.
Click the + button to add a new scheduler.
Fill in the following:
Name: UpdateCDNWhitelist
Start Time: Set the time for the first run.
Interval: Set to 1d for daily updates.
On Event: Type the name of your script: UpdateCDNWhitelist.
Click OK to save.

Mikrotik Licence status

در اکانت رسمی شرکت در سایت میکروتیک یک لایسنس chr خریداری شده است و با انجام عملیات renew licence در میکروتیک بروز می شود. این لایسنس محدوده زمانی ندارد و منقضی نخواد شد ولی نیاز به بروزرسانی دارد.






