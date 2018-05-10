#!/usr/bin/env bash
usage() {
    echo "Usage: $0 --cluster CLUSTER_NAME --service SERVICE_NAME --task TASK_NAME DOCKER_IMAGE"
    exit 1
}

while true ; do
    case "$1" in
        -t|--task) TASK_NAME=$2 ; shift 2 ;;
        -s|--service) SERVICE_NAME=$2 ; shift 2 ;;
        -c|--cluster) CLUSTER_NAME=$2 ; shift 2 ;;
        -h|--help) usage ;;
        --) shift ; break ;;
        *) break ;;
    esac
done

[ $# -eq 0 -o -z "$TASK_NAME" -o -z "$SERVICE_NAME" -o -z "$CLUSTER_NAME" ] && usage

DOCKER_IMAGE=$1

### Upgrade awscli
pip install --upgrade 'awscli==1.15.8'
source ~/.bashrc
###

### Set AWS Creds
aws configure set aws_access_key_id $AWS_ACCESS_KEY_ID
aws configure set aws_secret_access_key $AWS_SECRET_ACCESS_KEY
aws configure set default.region $AWS_REGION
aws configure set default.output json
###

##### Log in to aws
echo "Logging in"
eval $(aws ecr get-login --region $AWS_REGION --no-include-email)
######

echo "Get the previous task definition"
OLD_TASK_DEF=$(aws ecs describe-task-definition --task-definition $TASK_NAME --output json)
OLD_TASK_DEF_REVISION=$(echo $OLD_TASK_DEF | jq ".taskDefinition|.revision")

echo "dropping in the new image"
NEW_TASK_DEF=$(echo $OLD_TASK_DEF | jq --arg NDI $DOCKER_IMAGE '.taskDefinition.containerDefinitions[0].image=$NDI')

echo "create a new task template with all the required information to bring over"
FINAL_TASK=$(echo $NEW_TASK_DEF | jq '.taskDefinition|{family: .family, volumes: .volumes, containerDefinitions: .containerDefinitions}')

#Set variables for re-use
echo "Upload the task information and register the new task definition along with optional information"
UPDATED_TASK=$(aws ecs register-task-definition --cli-input-json "$(echo $FINAL_TASK)")
echo "Storing the Revision"
UPDATED_TASK_DEF_REVISION=$(echo $UPDATED_TASK | jq ".taskDefinition|.taskDefinitionArn")
echo "Updated task def revision: $UPDATED_TASK_DEF_REVISION"

echo "switch over to the new task definition by selecting the newest revision"
SUCCESS_UPDATE=$(aws ecs update-service --force-new-deployment --service $SERVICE_NAME --task-definition $TASK_NAME --cluster $CLUSTER_NAME)
