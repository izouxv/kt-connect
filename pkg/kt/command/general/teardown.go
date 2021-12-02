package general

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alibaba/kt-connect/pkg/common"
	"github.com/alibaba/kt-connect/pkg/kt"
	"github.com/alibaba/kt-connect/pkg/kt/cluster"
	"github.com/alibaba/kt-connect/pkg/kt/exec"
	"github.com/alibaba/kt-connect/pkg/kt/options"
	"github.com/alibaba/kt-connect/pkg/kt/registry"
	"github.com/alibaba/kt-connect/pkg/kt/util"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// CleanupWorkspace clean workspace
func CleanupWorkspace(cli kt.CliInterface, opts *options.DaemonOptions) {
	if !util.IsPidFileExist() {
		log.Info().Msgf("Workspace already cleaned")
		return
	}

	log.Info().Msgf("Cleaning workspace")
	cleanLocalFiles(opts)
	removePrivateKey(opts)
	if opts.RuntimeOptions.Component == common.ComponentConnect {
		recoverGlobalHostsAndProxy(opts)
		removeTunDevice(cli, opts)
	}

	ctx := context.Background()
	k8s, err := cli.Kubernetes()
	if err != nil {
		log.Error().Err(err).Msgf("Fails create kubernetes client when clean up workspace")
		return
	}
	if opts.RuntimeOptions.Component == common.ComponentExchange {
		recoverExchangedTarget(ctx, opts, k8s)
	} else if opts.RuntimeOptions.Component == common.ComponentMesh {
		recoverAutoMeshRoute(ctx, opts, k8s)
	}
	cleanService(ctx, opts, k8s)
	cleanShadowPodAndConfigMap(ctx, opts, k8s)
}

func removeTunDevice(cli kt.CliInterface, opts *options.DaemonOptions) {
	if opts.ConnectOptions.Method == common.ConnectMethodTun {
		log.Debug().Msg("Removing tun device ...")
		err := exec.RunAndWait(cli.Exec().Tunnel().RemoveDevice(), "del_device")
		if err != nil {
			log.Error().Err(err).Msgf("Fails to delete tun device")
		}

		if !opts.ConnectOptions.DisableDNS {
			err = util.RestoreConfig()
			if err != nil {
				log.Error().Err(err).Msgf("Restore resolv.conf failed")
			}
		}
	}
}

func recoverGlobalHostsAndProxy(opts *options.DaemonOptions) {
	if opts.RuntimeOptions.Dump2Host {
		log.Debug().Msg("Dropping hosts records ...")
		util.DropHosts()
	}
	if opts.ConnectOptions.UseGlobalProxy {
		log.Debug().Msg("Cleaning up global proxy and environment variable ...")
		if opts.ConnectOptions.Method == common.ConnectMethodSocks {
			registry.CleanGlobalProxy(&opts.RuntimeOptions.ProxyConfig)
		}
		registry.CleanHttpProxyEnvironmentVariable(&opts.RuntimeOptions.ProxyConfig)
	}
}

func cleanLocalFiles(opts *options.DaemonOptions) {
	pidFile := fmt.Sprintf("%s/%s-%d.pid", util.KtHome, opts.RuntimeOptions.Component, os.Getpid())
	if _, err := os.Stat(pidFile); err == nil {
		log.Info().Msgf("Removing pid %s", pidFile)
		if err = os.Remove(pidFile); err != nil {
			log.Error().Err(err).Msgf("Stop process %s failed", pidFile)
		}
	}

	jvmrcFilePath := util.GetJvmrcFilePath(opts.ConnectOptions.JvmrcDir)
	if jvmrcFilePath != "" {
		log.Info().Msg("Removing .jvmrc")
		if err := os.Remove(jvmrcFilePath); err != nil {
			log.Error().Err(err).Msgf("Delete .jvmrc failed")
		}
	}
}

func recoverExchangedTarget(ctx context.Context, opts *options.DaemonOptions, k cluster.KubernetesInterface) {
	if opts.ExchangeOptions.Method == common.ExchangeMethodScale && len(opts.RuntimeOptions.Origin) > 0 {
		log.Info().Msgf("Recovering origin deployment %s", opts.RuntimeOptions.Origin)
		err := k.ScaleTo(ctx, opts.RuntimeOptions.Origin, opts.Namespace, &opts.RuntimeOptions.Replicas)
		if err != nil {
			log.Error().Err(err).Msgf("Scale deployment %s to %d failed",
				opts.RuntimeOptions.Origin, opts.RuntimeOptions.Replicas)
		}
		// wait for scale complete
		ch := make(chan os.Signal)
		signal.Notify(ch, os.Interrupt, syscall.SIGINT)
		go func() {
			waitDeploymentRecoverComplete(ctx, opts, k)
			ch <- syscall.SIGINT
		}()
		_ = <-ch
	}
}

func recoverAutoMeshRoute(ctx context.Context, opts *options.DaemonOptions, k cluster.KubernetesInterface) {
	if opts.RuntimeOptions.Router != "" {
		routerPod, err := k.GetPod(ctx, opts.RuntimeOptions.Router, opts.Namespace)
		if err != nil {
			log.Error().Err(err).Msgf("Router pod has been removed unexpectedly")
			return
		}
		if shouldDelRouter, err2 := k.DecreaseRef(ctx, opts.RuntimeOptions.Router, opts.Namespace); err2 != nil {
			log.Error().Err(err2).Msgf("Decrease router pod %s reference failed", opts.RuntimeOptions.Shadow)
		} else if shouldDelRouter {
			recoverService(ctx, k, routerPod.Annotations[common.KtConfig], opts)
		} else {
			stdout, stderr, err3 := k.ExecInPod(common.DefaultContainer, opts.RuntimeOptions.Router, opts.Namespace,
				*opts.RuntimeOptions, common.RouterBin, "remove", opts.RuntimeOptions.Mesh)
			log.Debug().Msgf("Stdout: %s", stdout)
			log.Debug().Msgf("Stderr: %s", stderr)
			if err3 != nil {
				log.Error().Err(err3).Msgf("Failed to remove version %s from router pod", opts.RuntimeOptions.Mesh)
			}
		}
	}
}

func recoverService(ctx context.Context, k cluster.KubernetesInterface, routerConfig string, opts *options.DaemonOptions) {
	config := util.String2Map(routerConfig)
	svcName := config["service"]
	RecoverOriginalService(ctx, k, svcName, opts.Namespace)

	originSvcName := svcName + common.OriginServiceSuffix
	if err := k.RemoveService(ctx, originSvcName, opts.Namespace); err != nil {
		log.Error().Err(err).Msgf("Failed to remove origin service %s", originSvcName)
	}
	log.Info().Msgf("Substitution service %s removed", originSvcName)
}

func RecoverOriginalService(ctx context.Context, k cluster.KubernetesInterface, svcName, namespace string) {
	if svc, err := k.GetService(ctx, svcName, namespace); err != nil {
		log.Error().Err(err).Msgf("Original service %s not found", svcName)
		return
	} else {
		var selector map[string]string
		err = json.Unmarshal([]byte(svc.Annotations[common.KtSelector]), &selector)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to unmarshal original selector of service %s", svcName)
			return
		}
		svc.Spec.Selector = selector
		if _, err = k.UpdateService(ctx, svc); err != nil {
			log.Error().Err(err).Msgf("Failed to recover selector of original service %s (%s)",
				svcName, svc.Annotations[common.KtSelector])
		}
	}
	log.Info().Msgf("Original service %s recovered", svcName)
}

func waitDeploymentRecoverComplete(ctx context.Context, opts *options.DaemonOptions, k cluster.KubernetesInterface) {
	ok := false
	counts := opts.ExchangeOptions.RecoverWaitTime / 5
	for i := 0; i < counts; i++ {
		deployment, err := k.GetDeployment(ctx, opts.RuntimeOptions.Origin, opts.Namespace)
		if err != nil {
			log.Error().Err(err).Msgf("Cannot fetch original deployment %s", opts.RuntimeOptions.Origin)
			break
		} else if deployment.Status.ReadyReplicas == opts.RuntimeOptions.Replicas {
			ok = true
			break
		} else {
			log.Info().Msgf("Wait for deployment %s recover ...", opts.RuntimeOptions.Origin)
			time.Sleep(5 * time.Second)
		}
	}
	if !ok {
		log.Warn().Msgf("Deployment %s recover timeout", opts.RuntimeOptions.Origin)
	}
}

func cleanService(ctx context.Context, opts *options.DaemonOptions, k cluster.KubernetesInterface) {
	if opts.RuntimeOptions.Service != "" {
		log.Info().Msgf("Cleaning service %s", opts.RuntimeOptions.Service)
		err := k.RemoveService(ctx, opts.RuntimeOptions.Service, opts.Namespace)
		if err != nil {
			log.Error().Err(err).Msgf("Delete service %s failed", opts.RuntimeOptions.Service)
		}
	}
}

func cleanShadowPodAndConfigMap(ctx context.Context, opts *options.DaemonOptions, k cluster.KubernetesInterface) {
	shouldDelWithShared := false
	var err error
	if opts.RuntimeOptions.Shadow != "" {
		if opts.ConnectOptions != nil && opts.ConnectOptions.ShareShadow {
			shouldDelWithShared, err = k.DecreaseRef(ctx, opts.RuntimeOptions.Shadow, opts.Namespace)
			if err != nil {
				log.Error().Err(err).Msgf("Decrease shadow daemon pod %s ref count failed", opts.RuntimeOptions.Shadow)
			}
		} else {
			if opts.ExchangeOptions != nil && opts.ExchangeOptions.Method == common.ExchangeMethodEphemeral {
				for _, shadow := range strings.Split(opts.RuntimeOptions.Shadow, ",") {
					log.Info().Msgf("Removing ephemeral container of pod %s", shadow)
					err = k.RemoveEphemeralContainer(ctx, common.KtExchangeContainer, shadow, opts.Namespace)
					if err != nil {
						log.Error().Err(err).Msgf("Remove ephemeral container of pod %s failed", shadow)
					}
				}
			} else {
				for _, shadow := range strings.Split(opts.RuntimeOptions.Shadow, ",") {
					log.Info().Msgf("Cleaning shadow pod %s", shadow)
					err = k.RemovePod(ctx, shadow, opts.Namespace)
					if err != nil {
						log.Error().Err(err).Msgf("Delete shadow pod %s failed", shadow)
					}
				}
			}
		}
	}

	if opts.RuntimeOptions.SSHCM != "" && opts.ConnectOptions != nil && (shouldDelWithShared || !opts.ConnectOptions.ShareShadow) {
		for _, sshcm := range strings.Split(opts.RuntimeOptions.SSHCM, ",") {
			log.Info().Msgf("Cleaning configmap %s", sshcm)
			err = k.RemoveConfigMap(ctx, sshcm, opts.Namespace)
			if err != nil {
				log.Error().Err(err).Msgf("Delete configmap %s failed", sshcm)
			}
		}
	}
}

// removePrivateKey remove the private key of ssh
func removePrivateKey(opts *options.DaemonOptions) {
	if opts.RuntimeOptions.SSHCM == "" {
		return
	}
	for _, sshcm := range strings.Split(opts.RuntimeOptions.SSHCM, ",") {
		splits := strings.Split(sshcm, "-")
		component, version := splits[1], splits[len(splits)-1]
		file := util.PrivateKeyPath(component, version)
		if err := os.Remove(file); os.IsNotExist(err) {
			log.Error().Msgf("Key file %s not exist", file)
		}
	}
}