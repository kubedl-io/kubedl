## How to Add a Custom Deep Learning Workload

To facilitate extending KubeDL for more types of DL/ML workloads, we extract necessary APIs and interfaces in a common library.
This document describes how to extend KubeDL operator to support a custom workload type and build into a single binary.

1. Create a new folder kubedl/api/{workload_type} to import the definitions of your customized workload type, including 
   scheme registration as down below.
   
   ```go
   var (
       // SchemeGroupVersion is the group version used to register these objects.
       SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}
       // SchemeBuilder is used to add go types to the GroupVersionKind scheme
       SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
       // SchemeGroupVersionKind is the GroupVersionKind of the resource.
       SchemeGroupVersionKind = SchemeGroupVersion.WithKind(Kind)
   )
   ```
      
2. Create file `kdl/api/addtoscheme_{workload}_{version}` to append your scheme registration entry.

   ```go
   func init() {
       // SchemeBuilder defined in api/{workload_type}/{version}/register.go
       AddToSchemes = append(AddToSchemes, SchemeBuilder.AddToScheme)
   }
   ```

3. Create folder `kubedl/controllers/{workload}` where your workload controller is implemented. We summarize the necessary implementations as follows.

  - The reconciler must be initialized by manager in the `NewReconciler` function. That's how the reconciler can get access to meta dependencies such as Client, Cache, Scheme, etc.
  
  ```go
    // NewReconciler returns a new reconcile.Reconciler
    func NewReconciler(mgr ctrl.Manager, config job_controller.JobControllerConfiguration) reconcile.Reconciler {
    	r := &XXXReconciler{
    		Client: mgr.GetClient(),
    		scheme: mgr.GetScheme(),
    	}
    	r.recorder = mgr.GetEventRecorderFor(r.ControllerName())
    	// Initialize pkg job controller with components we only need.
    	r.ctrl = job_controller.JobController{
    		Controller:     r,
    		Expectations:   k8scontroller.NewControllerExpectations(),
    		Config:         config,
           // other fields...
    	}
        // your initializations...
    	return r
    }
  ```  
  
  - Set up with controller in the `SetupWithManager` function and specify which resources to watch  and how events handled.
  
  ```go
  func (r *XXXReconciler) SetupWithManager(mgr ctrl.Manager) error {
  	c, err := controller.New(r.ControllerName(), mgr, controller.Options{Reconciler: r})
  	if err != nil {
  		return err
  	}
  
  	// Watch owner resource with create event filter.
  	if err = c.Watch(&source.Kind{Type: &SomeWorkload{}}, &handler.EnqueueRequestForObject{}, predicate.Funcs{
  		CreateFunc: onOwnerCreateFunc(r),
  	}); err != nil {
  		return err
  	}
    
    // Watch other resources you concerned.
    // ...
  }
  ```
  
  - Implement interfaces in `kdl/pkg/job_controller/api/v1/interface.go`, which is indeed the detailed control logic of your custom workload.
  - Write your sync-reconcile logic in the `Reconcile` method, which will be called in controller runtime.
  
  ```go
  func (r *XXXReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
  	// Fetch the latest workload object from cluster.
  	obj := &SomeWorkload{}
  	err := r.Get(context.Background(), req.NamespacedName, obj)
  	
    // customized operations...

    // main entry of job reconciling.
  	err = r.ctrl.ReconcileJobs(tfJob, tfJob.Spec.TFReplicaSpecs, tfJob.Status, &tfJob.Spec.RunPolicy)
  	if err != nil {
  		return reconcile.Result{}, err
  	}
    
    // customized operations...
  	return reconcile.Result{}, err
  }
  ```
  
4. Create file `kdl/controllers/add_{workload}` to register your own controller to SetupWithManagerMap, so that your job controller will be 
   registered when the controller package is loaded.

  ```go
  func init() {
   SetupWithManagerMap["{workload}"] = func(mgr controllerruntime.Manager,  config job_controller.JobControllerConfiguration) error {
	   return NewReconciler(mgr, config).SetupWithManager(mgr)
   }
  }
  ```
 
5. Run `make manifests` to generate your CRD manifest file，which will be created under config/crd/bases.
6. Install your newly-generated manifest files to cluster by `kubectl apply -f config/crd/bases/{workload}.yaml`

Check for more details in the implementations for the currently supported workload types :)