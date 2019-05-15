package ingress

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cybozu-go/neco-ops/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func testExternalDNS() {
	It("should be deployed successfully", func() {
		Eventually(func() error {
			stdout, _, err := test.ExecAt(test.Boot0, "kubectl", "--namespace=ingress",
				"get", "deployment/external-dns", "-o=json")
			if err != nil {
				return err
			}

			deployment := new(appsv1.Deployment)
			err = json.Unmarshal(stdout, deployment)
			if err != nil {
				return err
			}

			if deployment.Status.AvailableReplicas != 1 {
				return fmt.Errorf("external-dns deployment's AvailableReplica is not 1: %d", int(deployment.Status.AvailableReplicas))
			}
			return nil
		}).Should(Succeed())
	})

	It("should create DNS record", func() {
		domainName := test.TestID + ".neco-ops.dev-ne.co"
		By("deploying DNSEndpoint")
		dnsEndpoint := fmt.Sprintf(`
apiVersion: externaldns.k8s.io/v1alpha1
kind: DNSEndpoint
metadata:
  name: dnsendpoint
  namespace: test-ingress
spec:
  endpoints:
  - dnsName: %s
    recordTTL: 180
    recordType: A
    targets:
    - 10.0.5.9
`, domainName)
		_, stderr, err := test.ExecAtWithInput(test.Boot0, []byte(dnsEndpoint), "kubectl", "apply", "-f", "-")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		By("resolving xxx.neco-ops.dev-ne.co")
		Eventually(func() error {
			stdout, stderr, err := test.ExecAt(test.Boot0, "kubectl", "get", "nodes", "-o", "json")
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}

			var nl corev1.NodeList
			err = json.Unmarshal(stdout, &nl)
			if err != nil {
				return err
			}
			if len(nl.Items) == 0 {
				return errors.New("nodes not found")
			}
			node := nl.Items[0]

			var nodeAddress string
			for _, addr := range node.Status.Addresses {
				if addr.Type != corev1.NodeInternalIP {
					continue
				}
				nodeAddress = addr.Address
			}
			if len(nodeAddress) == 0 {
				return errors.New("can not get NodeIP")
			}

			stdout, stderr, err = test.ExecAt(test.Boot0, "ckecli", "ssh", nodeAddress, "dig", "+noall", "+answer", domainName)
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			// expected: xxx.neco-ops.dev-ne.co. 300 IN A 10.0.5.9
			fields := strings.Fields(string(bytes.TrimSpace(stdout)))
			if len(fields) < 5 || fields[4] != "10.0.5.9" {
				return errors.New("expected IP address is 10.0.5.9, but actual response is " + string(stdout))
			}
			return nil
		}).Should(Succeed())
	})
}
