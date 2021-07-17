# Copyright 2018 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# ==============================================================================
"""An example of training Keras model with multi-worker strategies."""
from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import json
import os
import sys

import numpy as np
import tensorflow as tf
from tensorflow.contrib.saved_model.python.saved_model import keras_saved_model


def input_fn():
    x = np.random.random((1024, 10))
    y = np.random.randint(2, size=(1024, 1))
    x = tf.cast(x, tf.float32)
    dataset = tf.data.Dataset.from_tensor_slices((x, y))
    dataset = dataset.repeat(100)
    dataset = dataset.batch(32)
    return dataset


def _is_chief(task_type, task_id):
    # Note: there are two possible `TF_CONFIG` configuration.
    #   1) In addition to `worker` tasks, a `chief` task type is use;
    #      in this case, this function should be modified to
    #      `return task_type == 'chief'`.
    #   2) Only `worker` task type is used; in this case, worker 0 is
    #      regarded as the chief. The implementation demonstrated here
    #      is for this case.
    # For the purpose of this colab section, we also add `task_type is None`
    # case because it is effectively run with only single worker.
    return (task_type == 'worker' and task_id == 0) or task_type is None

def _get_temp_dir(dirpath, task_id):
    base_dirpath = 'workertemp_' + str(task_id)
    temp_dir = os.path.join(dirpath, base_dirpath)
    tf.io.gfile.makedirs(temp_dir)
    return temp_dir

def write_filepath(filepath, task_type, task_id):
    dirpath = os.path.dirname(filepath)
    base = os.path.basename(filepath)
    if not _is_chief(task_type, task_id):
        dirpath = _get_temp_dir(dirpath, task_id)
    return os.path.join(dirpath, base)

def main(args):
    if len(args) < 2:
        print('You must specify model_dir for checkpoints such as'
              ' /tmp/tfkeras_example/.')
        return

    model_dir = args[1]
    export_dir = args[2]
    print('Using %s to store checkpoints.' % model_dir)

    # Define a Keras Model.
    model = tf.keras.Sequential()
    model.add(tf.keras.layers.Dense(16, activation='relu', input_shape=(10,)))
    model.add(tf.keras.layers.Dense(1, activation='sigmoid'))

    # Compile the model.
    optimizer = tf.train.GradientDescentOptimizer(0.2)
    model.compile(loss='binary_crossentropy', optimizer=optimizer)
    model.summary()
    tf.keras.backend.set_learning_phase(True)

    # Define DistributionStrategies and convert the Keras Model to an
    # Estimator that utilizes these DistributionStrateges.
    # Evaluator is a single worker, so using MirroredStrategy.
    config = tf.estimator.RunConfig(
        experimental_distribute=tf.contrib.distribute.DistributeConfig(
            train_distribute=tf.contrib.distribute.CollectiveAllReduceStrategy(
                num_gpus_per_worker=0),
            eval_distribute=tf.contrib.distribute.MirroredStrategy(
                num_gpus_per_worker=0)))
    keras_estimator = tf.keras.estimator.model_to_estimator(
        keras_model=model, config=config, model_dir=model_dir)

    # Train and evaluate the model. Evaluation will be skipped if there is not an
    # "evaluator" job in the cluster.
    tf.estimator.train_and_evaluate(
        keras_estimator,
        train_spec=tf.estimator.TrainSpec(input_fn=input_fn),
        eval_spec=tf.estimator.EvalSpec(input_fn=input_fn))
    # def serving_input_receiver_fn():
    #     inputs = {model.input_names[0]: tf.placeholder(tf.float32, [None, 28, 28])}
    #     return tf.estimator.export.ServingInputReceiver(inputs, inputs)
    #
    # keras_estimator.export_saved_model('saved_model', serving_input_receiver_fn)
    # task_type, task_id = (strategy.cluster_resolver.task_type,
    #                       strategy.cluster_resolver.task_id)

    tf_config_json = os.environ.get("TF_CONFIG", "{}")
    tf_config = json.loads(tf_config_json)
    task = tf_config.get("task", {})
    job_name = task["type"]
    task_id = task["index"]

    is_chief = (job_name == 'chief' or job_name == 'worker') and (task_id == 0)
    if is_chief:
        print("exporting model ")
        # only save the model for chief
        keras_saved_model.save_keras_model(model, export_dir)
        print("exported model ")


if __name__ == '__main__':
    tf.logging.set_verbosity(tf.logging.INFO)
    tf.app.run(argv=sys.argv)