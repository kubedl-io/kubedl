
export const isGit = (info) => {
  if (info.metadata.annotations && info.metadata.annotations['kubedl.io/git-sync-config']) {
    return true;
  }
  return false;
};

export const getCommand = (info) => {
  const containersInfo = info?.kind === 'TFJob' ? Object.values(info?.spec?.tfReplicaSpecs)[0]?.template?.spec?.containers[0]
    : Object.values(info?.spec?.pytorchReplicaSpecs)[0]?.template?.spec?.containers[0];
  const regExp = /^prepare_dlc_environment\s&&\s/g;
  const command = containersInfo?.command ?? [];
  const inputValue = command.reverse()[0];
  if (regExp.test(inputValue)) {
    return inputValue.replace(regExp, '');
  }
  if (command.includes('/bin/sh')) {
    return inputValue;
  }
  return containersInfo.command.toString();
};

export const getRequirementsDir = (info) => {
  const envInfo = info?.kind === 'TFJob' ? Object.values(info?.spec?.tfReplicaSpecs)[0]?.template?.spec?.containers[0]?.env
    : Object.values(info?.spec?.pytorchReplicaSpecs)[0]?.template?.spec?.containers[0]?.env;
  return {
    enabled: envInfo ? envInfo.some(n => n.name === 'REQUIREMENTS_DIR') : false,
    requirementsDir: envInfo && envInfo.some(n => n.name === 'REQUIREMENTS_DIR') ? envInfo.filter(n => n.name === 'REQUIREMENTS_DIR')[0].value : '',
  };
};

export const getInfoPvc = (info) => {
  if (info?.kind === 'TFJob') {
    const pvcInfo = Object.values(info?.spec?.tfReplicaSpecs)[0]?.template?.spec?.volumes[0];
    return pvcInfo.name;
  }
  return Object.values(info?.spec?.pytorchReplicaSpecs)[0]?.template?.spec?.volumes[0]?.name;
};

export const getTensorboard = (info) => {
  const tensorboard = info?.metadata?.annotations?.['kubedl.io/tensorboard-config'];
  return {
    enabled: !!tensorboard,
    logDir: eval(`(${tensorboard})`)?.logDir ?? '',
  };
};

export const getTasks = (info) => {
  const taskInfo = info?.kind === 'TFJob' ? info?.spec?.tfReplicaSpecs : info?.spec?.pytorchReplicaSpecs;
  const newTask = [];
  for (const i in taskInfo) {
    const taskObj = {
      role: i === 'Chief' || i === 'Master' ? 'Worker' : i,
      replicas: info?.kind !== 'TFJob' && i === 'Worker' ? taskInfo[i]?.replicas + 1 : taskInfo[i]?.replicas,
      resource: {
        gpu: taskInfo[i]?.template?.spec?.containers?.[0]?.resources?.limits?.['nvidia.com/gpu'] ?? 0,
        cpu: taskInfo[i]?.template?.spec?.containers?.[0]?.resources?.limits?.cpu ?? 4,
        memory: Number(taskInfo[i]?.template?.spec?.containers?.[0]?.resources?.limits?.memory?.split('Gi')[0]) ?? 8,
      },
      image: taskInfo[i]?.template?.spec?.containers?.[0]?.image ?? tfJobImages[0],
    };
    if (i === 'Worker') {
      newTask.unshift(taskObj);
    } else {
      newTask.push(taskObj);
    }
  }
  return duplicateRemoval(newTask);
};

export const getEnv = (info) => {
  const envInfo = info?.kind === 'TFJob' ? Object.values(info?.spec?.tfReplicaSpecs)[0] : Object.values(info?.spec?.pytorchReplicaSpecs)[0];
  const envList = envInfo?.template?.spec?.containers?.[0]?.env ?? [];
  const newEnv = JSON.parse(JSON.stringify(envList) || '{}');
  return newEnv.filter(e => e.name !== 'REQUIREMENTS_DIR' && e.name !== 'DLC_USER_TOKEN');
};

export const getMount = (info) => {
  const mountInfo = info?.kind === 'TFJob' ? Object.values(info?.spec?.tfReplicaSpecs)[0] : Object.values(info?.spec?.pytorchReplicaSpecs)[0];
  const mountList = mountInfo?.template?.spec?.containers?.[0]?.volumeMounts ?? [];
  const newMountList = [];
  mountList.length > 0 && mountList.map((v) => {
    newMountList.push({
      pvc: v.name,
      path: v.mountPath,
    });
  });
  return newMountList.filter(v => v.path !== '/workspace');
};

export const duplicateRemoval = (task) => {
  const obj = {};
  task = task.reduce((item, next) => {
    obj[next.role] ? '' : obj[next.role] = true && item.push(next);
    return item;
  }, []);
  return task;
};

export const getCookie = (name) => {
  const match = document.cookie.match(new RegExp(`(^| )${name}=([^;]+)`));
  if (match) return match[2];
};

export const handleDataSource = (info) => {
  const dataInfo = info?.kind === 'TFJob' ? Object.values(info?.spec?.tfReplicaSpecs)[0]?.template?.spec
    : Object.values(info?.spec?.pytorchReplicaSpecs)[0]?.template?.spec;
  return (dataInfo.volumes || []).map(({ name }) => ({ dataSource: name.replace(/^data-/g, '') }));
};

export const handleCodeSource = (info) => {
  if (isGit(info)) {
    const gitInfo = JSON.parse(info?.metadata?.annotations?.['kubedl.io/git-sync-config']);
    if (gitInfo.aliasName && gitInfo.aliasName.indexOf('code-') != -1) {
      return gitInfo.aliasName.slice(5);
    }
  } else {
    const codeInfo = info?.kind === 'TFJob' ? Object.values(info?.spec?.tfReplicaSpecs)[0]?.template?.spec
      : Object.values(info?.spec?.pytorchReplicaSpecs)[0]?.template?.spec;
    if (codeInfo.volumes) {
      if (codeInfo.volumes.some(v => v.name.indexOf('code-') != -1)) {
        return codeInfo.volumes.filter(v => v.name.indexOf('code-') != -1)[0].name.slice(5);
      }
    }
    return null;
  }
};

export const handleRequirementsData = (info) => {
  const requirementsInfo = info?.kind === 'TFJob' ? Object.values(info?.spec?.tfReplicaSpecs)[0]?.template?.spec?.containers[0]?.env
    : Object.values(info?.spec?.pytorchReplicaSpecs)[0]?.template?.spec?.containers[0]?.env;
  return {
    enabled: requirementsInfo ? requirementsInfo.some(n => n.name === 'REQUIREMENTS_DIR') ? 'catalog' : 'textBox' : 'textBox',
    text: requirementsInfo && requirementsInfo.some(n => n.name === 'REQUIREMENTS_TEXT')
      ? requirementsInfo.filter(n => n.name === 'REQUIREMENTS_TEXT')[0].value.replace(/[,]/g, '\n') : '',
    catalog: requirementsInfo && requirementsInfo.some(n => n.name === 'REQUIREMENTS_DIR')
      ? requirementsInfo.filter(n => n.name === 'REQUIREMENTS_DIR')[0].value : '',
  };
};

export const handleKindType = (info) => {
  const KindInfo = info?.kind === 'TFJob' ? info?.spec?.tfReplicaSpecs : info?.spec?.pytorchReplicaSpecs;
  let isStandAlone = false;
  isStandAlone = Object.keys(KindInfo).every(v => v === 'Chief') && KindInfo.Chief.replicas === 1
      || Object.keys(KindInfo).every(v => v === 'Worker') && KindInfo.Worker.replicas === 1
      || Object.keys(KindInfo).every(v => v === 'Master') && KindInfo.Master.replicas === 1;
  if (isStandAlone) {
    return info?.kind === 'TFJob' ? 'TFJob' : 'PyTorchJob';
  }
  return info?.kind === 'TFJob' ? 'TFJobDistributed' : 'PyTorchJobDistributed';
};

export const handleCodeSourceBranch = (info) => {
  if (isGit(info)) {
    return JSON.parse(info?.metadata?.annotations?.['kubedl.io/git-sync-config']).branch;
  }
  return '';
};

export const handleWorkingDir = (info) => {
  const workingDirInfo = info?.kind === 'TFJob' ? Object.values(info?.spec?.tfReplicaSpecs)[0]?.template?.spec?.containers[0]?.workingDir
    : Object.values(info?.spec?.pytorchReplicaSpecs)[0]?.template?.spec?.containers[0]?.workingDir;
  return workingDirInfo;
};
