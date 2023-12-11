export class Iterator {
    //todo look up better way of doing an async iterator in js
    constructor(data,path,config,page){
        this.config = config
        this.path = path
        if (data !== null){
            this.responseData = new IteratorResponseData(data)
        }else{
            this.responseData = null
        }

        this.currentPages = []
        this.current=null
        this.err = null
        this.message = ""

        this.offset = 0
        this.singlePage=false
        this.currentItem = 0
        if (page !== null){
            this.pagination = page
        }else{
            this.pagination = new Pagination(null)
        }

    }

    /**
     *
     * @param {int} itemsPerPage
     * @constructor
     */
    async SetItemsPerPage(itemsPerPage){
        if (itemsPerPage > 500 || itemsPerPage < 1){
            return
        }
        this.pagination.ItemsPerPage = itemsPerPage
        return new Promise(resolve => resolve(this.currentPages))
    }

    async GetCurrent(){
        if (this.current == null){
            if (this.currentPages === null || this.currentPages.length === 0){
                let v = await this.getPages()
                if (!v) {
                    return new Promise((resolve, reject) => reject(`failed loading pages ${this.err}`))
                }
                if (!Array.isArray(this.currentPages)){
                    this.singlePage = true
                    this.current = this.currentPages
                    return new Promise(resolve => resolve(this.current))
                }
            }
            if (this.currentItem - this.offset >= this.currentPages.length){
                return new Promise((resolve, reject)=> reject(`index out of bounds ${this.currentItem - this.offset} > ${this.currentPages.length}`))
            }
            this.current = this.currentPages[this.currentItem - this.offset ]
        }
        return new Promise(resolve => resolve(this.current))
    }

    /**
     * @return {promise<array>|promise<null>}
     */
    async GetPage(){
        if (this.currentPages === null || this.currentPages.length === 0){
            let v = await this.getPages()
            if (!v) {
                return new Promise((resolve, reject) => reject(`failed loading pages ${this.err}`))
            }
        }
        if (!Array.isArray(this.currentPages)){
            this.singlePage = true
            this.current = this.currentPages
            return new Promise(resolve => resolve(this.current))
        }
        return new Promise(resolve => resolve(this.currentPages))
    }

    /**
     * @param {int} pageNumber
     * @returns {promise<array>}
     * @constructor
     */
    async GoToPage(pageNumber){
        if (this.singlePage) {
            return null
        }
        this.pagination.CurrentPage = pageNumber
        if (this.pagination.CurrentPage < 1){
            this.pagination.CurrentPage = 1
        }
        if (this.pagination.CurrentPage > this.pagination.TotalPages){
            this.pagination.CurrentPage = this.pagination.TotalPages
        }
        let v = await this.getPages()
        if (!v) {
            return new Promise((resolve, reject) => reject(`failed loading pages ${this.err}`))
        }
        return new Promise(resolve => resolve(this.currentPages))
    }

    /**
     *
     * @return {promise<array>}
     * @constructor
     */
    async PreviousPage(){
        if (this.singlePage) {
            return null
        }
        this.pagination.CurrentPage -= 1
        if (this.pagination.CurrentPage < 1){
            this.pagination.CurrentPage = 1
        }
        let v = await this.getPages()
        if (!v) {
            return new Promise((resolve, reject) => reject(`failed loading pages ${this.err}`))
        }
        return new Promise(resolve => resolve(this.currentPages))
    }

    /**
     *
     * @return {promise<array>}
     * @constructor
     */
    async NextPage(){
        if (this.singlePage) {
            return null
        }
        this.pagination.CurrentPage += 1
        if (this.pagination.CurrentPage < 1){
            this.pagination.CurrentPage = 1
        }
        if (this.pagination.CurrentPage > this.pagination.CurrentPage){
            this.pagination.CurrentPage = this.pagination.TotalPages
        }
        let v = await this.getPages()
        if (!v) {
            return new Promise((resolve, reject) => reject(`failed loading pages ${this.err}`))
        }
        return new Promise(resolve => resolve(this.currentPages))

    }
    /**
     *
     * @returns {promise}
     * @constructor
     */
    async Next(){
        if (this.singlePage) {
            return null
        }
        if (this.pagination.TotalItems === 0){
            let v = await this.getPages()
            if (!v) {
                return new Promise((resolve, reject) => reject(`failed loading pages ${this.err}`))
            }
            if (!Array.isArray(this.currentPages)){
                this.singlePage = true
                this.current = this.currentPages
                return new Promise(resolve => resolve(this.current))
            }
            //todo check if it an array
            if (this.currentPages.length === 0){
                return null
            }
            this.current = this.currentPages[this.currentItem - this.offset]
            return new Promise(resolve => resolve(this.current))
        }
        if (this.currentItem < this.pagination.TotalItems) {
            this.currentItem += 1
            if (this.currentItem-this.offset >= this.currentPages.length){
                let v = await this.getPages()
                if (!v) {
                    return new Promise((resolve, reject) => reject(`failed loading pages ${this.err}`))
                }
            }
            if (this.currentItem-this.offset >= this.currentPages.length){
                return null
            }
            this.current = this.currentPages[this.currentItem - this.offset]
            return new Promise(resolve => resolve(this.current))
        }
        return null
    }

    Err(){
        return this.err
    }

    /**
     *
     * @returns {promise<string>}
     * @constructor
     */
    async Message(){
        await this.responseData.LoadData()
        if (this.message === null || this.message === ""){
            if (this.responseData !== null){
                return new Promise(resolve => resolve(this.responseData.Message))
            }else{
                let v = await this.getPages()
                if (!v) {
                    return new Promise((resolve, reject) => reject(`failed loading pages ${this.err}`))
                }
                return this.responseData.Message
            }

        }
        return new Promise(resolve => resolve(this.message))
    }

    /**
     *
     * @returns {promise<boolean>}
     */
    async getPages(){
        await this.responseData.LoadData()
        console.log(this.responseData)
        //todo or current page is greater than what we have
        if (this.responseData.Data === undefined || this.responseData.Data === null){
            this.config.params["items_per_page"] = this.pagination.ItemsPerPage
            this.config.params["page"] = this.pagination.CurrentPage
            try {
                const data = await $fetch(this.path, this.config)
                this.responseData = new IteratorResponseData(data)
                await this.responseData.LoadData()
            }catch (error){
                this.err = error
                this.message = error.Message
                return new Promise(resolve => resolve(false),reject=>reject(error))
            }
        }
        if (!(this.responseData.Page === undefined || this.responseData.Page === null)){
            this.pagination = this.responseData.Page
        }
        this.message = this.responseData.Message
        this.currentPages = this.responseData.Data

        if (!Array.isArray(this.currentPages)){
            this.singlePage = true
            this.current = this.currentPages
            return new Promise(resolve => resolve(true))
        }
        this.offset = (this.pagination.CurrentPage - 1) * this.pagination.ItemsPerPage
        return new Promise(resolve => resolve(true))
    }

}

class IteratorResponseData {
    constructor(rawResponse) {
        this.rawResponse = rawResponse
        this.decoded = {}
        if (rawResponse === undefined){
            return
        }
        if ("data" in rawResponse) {
            this.Data = rawResponse.data
        }else{
            this.Data = []
        }
        if ("page" in rawResponse) {
            this.Page = new Pagination(rawResponse["page"])
        }

        if ("message" in rawResponse) {
            this.Message = rawResponse.message
        }
    }
    async LoadData() {
        if (this.rawResponse === undefined){
            return
        }
        console.log(this.rawResponse)
        const rawData = await this.rawResponse
        this.decoded = JSON.parse(rawData)
        this.parse()
    }
    parse(){
        if ("data" in this.decoded) {
            this.Data = this.decoded.data
        }
        if ("page" in this.decoded) {
            this.Page = new Pagination(this.decoded["page"])
        }
        if ("message" in this.decoded) {
            this.Message = this.decoded.message
        }
    }
}

class Pagination {
    constructor(pageJson) {
        if (pageJson === null || pageJson === undefined){
            this.CurrentPage = 1
            this.ItemsPerPage = 24
            return
        }
        this.CurrentPage = pageJson["current_page"]
        this.NextPage = pageJson["next_page"]
        this.TotalItems = pageJson["total_items"]
        this.TotalPages = pageJson["total_pages"]
        this.ItemsPerPage = pageJson["items_per_page"]
    }
}