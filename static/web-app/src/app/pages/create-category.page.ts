import { CommonModule } from '@angular/common';
import { Component } from '@angular/core';
import { RouterModule } from '@angular/router';

@Component({
  selector: 'app-create-category',
  templateUrl: './create-category.page.html',
  imports: [CommonModule, RouterModule],
})
export class CreateCategoryPage {
  constructor() {}
}
