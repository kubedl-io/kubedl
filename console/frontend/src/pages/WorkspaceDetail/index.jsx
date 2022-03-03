import {Component} from "react";
import JobTableList from "@/pages/Jobs";
import NotebookTableList from "@/pages/Notebooks";
import {Divider} from "antd";

class WorkspaceDetail extends Component {
    render() {
        const search = this.props.location.search;
        const name = new URLSearchParams(search).get("name");
        // workspace name is the same as namespace

        return (
            <React.Fragment>

                <JobTableList namespace={name}/>
                <Divider/>
                <NotebookTableList namespace={name}/>
            </React.Fragment>
    );
    }
}

export default WorkspaceDetail
