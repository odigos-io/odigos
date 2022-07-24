package controllers

//func (r *CollectorReconciler) syncConfigMaps(ctx context.Context, collector *odigosv1.Collector) (bool, error) {
//	cmList, err := r.listConfigMaps(ctx, collector)
//	if err != nil {
//		return false, err
//	}
//
//	destList, err := r.listDestinations(ctx, collector)
//	if err != nil {
//		return false, err
//	}
//
//	cmData, err := collectorconfig.GetConfigForCollector(destList)
//	if err != nil {
//		return false, err
//	}
//
//	if len(cmList.Items) == 0 {
//		err = r.createConfigMap(ctx, collector, cmData)
//		if err != nil {
//			return false, err
//		}
//		return true, nil
//	}
//
//	if !r.isConfigMapUpToDate(cmList, cmData) {
//		err = r.updateConfigMaps(ctx, cmList, cmData)
//		if err != nil {
//			return false, err
//		}
//
//		return true, nil
//	}
//
//	return false, nil
//}
//
//func (r *CollectorReconciler) isConfigMapUpToDate(cmList *v1.ConfigMapList, cmData string) bool {
//	for _, cm := range cmList.Items {
//		if curData, exists := cm.Data["collector-conf"]; !exists || curData != cmData {
//			return false
//		}
//	}
//
//	return true
//}
//
//func (r *CollectorReconciler) updateConfigMaps(ctx context.Context, cmList *v1.ConfigMapList, cmData string) error {
//	for _, cm := range cmList.Items {
//		cm.Data["collector-conf"] = cmData
//		err := r.Update(ctx, &cm)
//		if err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
//
//func (r *CollectorReconciler) createConfigMap(ctx context.Context, collector *odigosv1.Collector, cmData string) error {
//	configmap := &v1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      collector.Name,
//			Namespace: collector.Namespace,
//		},
//		Data: map[string]string{
//			"collector-conf": cmData,
//		},
//	}
//
//	err := ctrl.SetControllerReference(collector, configmap, r.Scheme)
//	if err != nil {
//		return err
//	}
//
//	err = r.Create(ctx, configmap)
//	if err != nil {
//		if apierrors.IsAlreadyExists(err) {
//			return nil
//		}
//		return err
//	}
//
//	return nil
//}
//
//func (r *CollectorReconciler) listConfigMaps(ctx context.Context, collector *odigosv1.Collector) (*v1.ConfigMapList, error) {
//	var cmList v1.ConfigMapList
//	err := r.List(ctx, &cmList, client.InNamespace(collector.Namespace), client.MatchingFields{ownerKey: collector.Name})
//	if err != nil {
//		return nil, err
//	}
//
//	return &cmList, nil
//}
//
//func (r *CollectorReconciler) listDestinations(ctx context.Context, collector *odigosv1.Collector) (*odigosv1.DestinationList, error) {
//	var destList odigosv1.DestinationList
//	err := r.List(ctx, &destList, client.InNamespace(collector.Namespace))
//	if err != nil {
//		return nil, err
//	}
//
//	return &destList, nil
//}
